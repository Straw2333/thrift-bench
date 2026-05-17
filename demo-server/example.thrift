namespace go example

// 用户信息结构
struct User {
    1: i64 id,
    2: string name,
    3: i32 age,
    4: string email
}

// 获取用户请求
struct GetUserRequest {
    1: i64 userId
}

// 获取用户响应
struct GetUserResponse {
    1: User user,
    2: i32 code,
    3: string message
}

// 用户服务
service UserService {
    // 获取用户信息
    GetUserResponse getUser(1: GetUserRequest req)

    // 测试连接（无参数）
    string ping()
}
