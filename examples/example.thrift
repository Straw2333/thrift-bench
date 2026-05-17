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

// 创建用户请求
struct CreateUserRequest {
    1: string name,
    2: i32 age,
    3: string email
}

// 创建用户响应
struct CreateUserResponse {
    1: User user,
    2: i32 code,
    3: string message
}

// 用户服务
service UserService {
    // 获取用户信息
    GetUserResponse getUser(1: GetUserRequest req)

    // 创建用户
    CreateUserResponse createUser(1: CreateUserRequest req)

    // 测试连接（无参数）
    string ping()

    /**
     * 获取License信息
     */
    string getLicenseInfo(
        /**
         * 用户登录 token
         */
        1: required string token,

        /**
         * 设备ID 如果是空字符串，则操作本设备
         */
        2: required string device)

    i32 ceshi
    (
        1: i32 iRet,
        2: string test)
}
