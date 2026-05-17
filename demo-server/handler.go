package main

import (
	"context"
	"log"

	example "demo-server/kitex_gen/example"
)

// UserServiceImpl implements the last service interface defined in the IDL.
type UserServiceImpl struct{}

// GetUser implements the UserServiceImpl interface.
func (s *UserServiceImpl) GetUser(ctx context.Context, req *example.GetUserRequest) (resp *example.GetUserResponse, err error) {
	log.Printf("收到请求: GetUser, userId=%d", req.UserId)

	// 模拟返回用户信息
	user := &example.User{
		Id:    req.UserId,
		Name:  "张三",
		Age:   25,
		Email: "zhangsan@example.com",
	}

	resp = &example.GetUserResponse{
		User:    user,
		Code:    200,
		Message: "success",
	}

	return resp, nil
}

// Ping implements the UserServiceImpl interface.
func (s *UserServiceImpl) Ping(ctx context.Context) (resp string, err error) {
	log.Println("收到请求: Ping")
	return "pong", nil
}

