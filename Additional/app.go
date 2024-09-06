package main

import (
	"fmt"
	"net"
	"net/http"
	"context"
	"os"
	"strconv"
	"strings"
	"encoding/base64"
	"encoding/json"
	"time"
	qrcode  "github.com/skip2/go-qrcode" 
	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
)


// 全局变量，用于存储 Snowflake 节点实例
var node *snowflake.Node
var (
	ctx = context.Background()
	rdb *redis.Client
)


// Claims 结构体
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 定义用于签名 JWT 的密钥
var jwtKey = []byte("80BEB12D58BC822705B6000584249652")

var url string = "http://"+ getHostIp() + ":8080"

func init() {
	// 初始化 Redis 客户端
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 服务器地址
		Password: "",               // Redis 密码
		DB:       0,                // 数据库编号
	})
}

func readHTMLFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loginHandler 处理 /login 路由
func loginHandler(w http.ResponseWriter, r *http.Request) {
	content, err := readHTMLFile("login.html")
	if err != nil {
		http.Error(w, "Could not read login.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}

func mobileHandler(w http.ResponseWriter, r *http.Request) {
	content, err := readHTMLFile("MobileVerify.html")
	if err != nil {
		http.Error(w, "Could not read MobileVerify.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		uuid = node.Generate().String() // 在 Redis 查询发生错误时生成新的 UUID
	} else {
		expireTimestamp := parts[2]
		if isTimestampExpired(expireTimestamp) {
			err = rdb.Del(ctx, uuid).Err()
			if err != nil {
				return
			}
			uuid = node.Generate().String() // 在过期时生成新的 UUID
		}
	}

	qrcodeURL := url + "/MobileVerify?uuid=" + uuid

	// 生成二维码，返回一个字节数组
	png, err := qrcode.Encode(qrcodeURL, qrcode.Medium, 256)
	if err != nil {
		fmt.Println("无法生成二维码", err)
		return
	}

	// 将二维码字节数组转换为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(png)

	// 使用结构体来保证字段顺序
	data := struct {
		UUID   string `json:"uuid"`
		QRCode string `json:"qrcode"`
	}{
		UUID:   uuid,
		QRCode: base64Str,
	}

	// 将结构体转换为 JSON 字符串
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	content := string(jsonData)

	storeDataInRedis(uuid)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}


func qrcodeLoginHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	userid := r.URL.Query().Get("userid")
	
	var content string

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		content = `{"status": "fail","msg": "uuid has expired"}`
	} else {
		expireTimestamp := parts[2]
		if isTimestampExpired(expireTimestamp) {
			err = rdb.Del(ctx, uuid).Err()
			if err != nil {
				return
			}
			content = `{"status": "fail","msg": "uuid has expired"}` 
			err = rdb.Del(ctx, uuid).Err()
			if err != nil {
				return
			}
		} else {
			redisTimestamp := parts[2]

			dataToRedis := fmt.Sprintf("%s-%s-%s", uuid, userid, redisTimestamp)
		
			// 将数据写入 Redis
			redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
			if redisErr != nil {
				fmt.Println("Error setting value in Redis:", redisErr)
				return
			}
		
		
			content = `{"status": "ok"}`
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")

	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		content := `{"status": "fail", "msg": "uuid has expired"}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(content))
		return
	}

	var content string
	if parts[1] != "0" {
		token := login(parts[1])
		content = `{"status": "ok", "token": "` + token + `"}`
	} else {
		content = `{"status": "fail", "msg": "userid not received"}`
	}

	expireTimestamp := parts[2]

	if isTimestampExpired(expireTimestamp) {
		content = `{"status": "fail", "msg": "uuid has expired"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	var content string
	if validateToken(token) {
		content = `{"status": "ok"}`
	} else {
		content = `{"status": "fail"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func getUuidHandler(w http.ResponseWriter, r *http.Request) {
	uuid := node.Generate().String()

	content := `{"uuid": ` + uuid + `}`

	storeDataInRedis(uuid)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func validateUuidHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	
	parts, err := getParsedDataFromRedis(uuid)
	if err != nil {
		fmt.Println(err)
		return
	}

	expireTimestamp := parts[2]

	var content string
	if isTimestampExpired(expireTimestamp) {
		content = `{"status": "fail"}`
		err = rdb.Del(ctx, uuid).Err()
		if err != nil {
			return
		}
	} else {
		content = `{"status": "ok"}`
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func isTimestampExpired(expireTimestamp string) (bool) {
	expireTime, err := strconv.ParseInt(expireTimestamp, 10, 64)
	if err != nil {
		return false
	}

	// 获取当前时间的Unix时间戳（以秒为单位）
	currentTime := time.Now().Unix()

	// 比较两个时间戳
	if currentTime > expireTime {
		return true
	} else {
		return false
	}
}

func main() {
	var err error
	// 创建 Snowflake 节点实例
	node, err = snowflake.NewNode(1)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/MobileVerify", mobileHandler)

	http.HandleFunc("/do/generateQRCode", qrcodeHandler)
	http.HandleFunc("/do/qrcodeLogin", qrcodeLoginHandler)
	http.HandleFunc("/do/getStatus", statusHandler)
	http.HandleFunc("/do/validateToken", validateHandler)
	http.HandleFunc("/do/getUuid", getUuidHandler)
	http.HandleFunc("/do/validateUuid", validateUuidHandler)

	// 启动服务器，监听在 8080 端口
	fmt.Println("Starting server at port 8080,you can use "+ url + "/login to visit.")
	httpErr := http.ListenAndServe(":8080", nil)
	if httpErr != nil {
		fmt.Println("Error starting server:", httpErr)
	}

}

func login(userid string) string {
	// 由于这里并没有用户表，所以不进行过滤，任何userid都可以登录

	// 创建 token
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: userid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		fmt.Println("生成 token 时出错:", err)
		return ""
	}

	return tokenString
}

func validateToken(tokenString string) bool {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false
		}
		return false
	}

	if !token.Valid {
		fmt.Println("无效的 token")
		return false
	}

	return true
}

func getHostIp() string {
	conn, err := net.Dial("udp", "119.29.29.29:53")
	if err != nil {
		fmt.Println("get current host ip err:", err)
		return ""
	}
	defer conn.Close() 

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		fmt.Println("failed to get UDP address")
		return ""
	}

	ip := strings.Split(addr.String(), ":")[0]
	return ip
}

func getParsedDataFromRedis(uuid string) ([]string, error) {
	// 从 Redis 中读取数据
	val, err := rdb.Get(ctx, uuid).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting value from Redis: %w. It is possible that the UUID has expired or is invalid", err)
	}

	// 解析 Redis 中的数据
	parts := strings.Split(val, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid data format in Redis")
	}

	return parts, nil
}

func storeDataInRedis(uuid string) error {
	// 当前时间戳
	timestamp := time.Now().Unix() + 30

	// 拼接数据
	dataToRedis := fmt.Sprintf("%s-%s-%d", uuid, "0", timestamp)

	// 将数据写入 Redis
	redisErr := rdb.Set(ctx, uuid, dataToRedis, 0).Err()
	if redisErr != nil {
		return fmt.Errorf("error setting value in Redis: %w", redisErr)
	}

	return nil
}