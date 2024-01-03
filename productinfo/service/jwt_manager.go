package main

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "productinfo/service/ecommerce"
	"time"
)

type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

type UserClaims struct {
	jwt.StandardClaims
	Username string `json:"username"`
	Role     string `json:"role"`
}

type AuthServer struct {
	userStore  UserStore
	jwtManager *JWTManager
	pb.UnimplementedAuthServiceServer
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey, tokenDuration}
}

func (manager *JWTManager) Generate(user *User) (string, error) {
	claims := UserClaims{
		StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(manager.tokenDuration).Unix()},
		Username:       user.Username,
		Role:           user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(manager.secretKey))
}

func (manager *JWTManager) Verify(accessToken string) (*UserClaims, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("unexpected token signing method")
		}
		return []byte(manager.secretKey), nil
	}
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		keyFunc,
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func NewAuthServer(userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{userStore, jwtManager, pb.UnimplementedAuthServiceServer{}}
}

func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := server.userStore.Find(req.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot find user: %v", err)
	}

	if user == nil || !user.IsCorrectPassword(req.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	token, err := server.jwtManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	res := &pb.LoginResponse{AccessToken: token}
	return res, nil
}
