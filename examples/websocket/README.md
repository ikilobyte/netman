## 测试用例
> 使用 [crossbario/autobahn-testsuite](https://github.com/crossbario/autobahn-testsuite)


## 准备配置文件
* 请参考 [config/fuzzingclient.json](./config/fuzzingclient.json)

## 服务器响应
* 收到什么回复什么即可，不要额外回复其它的信息，即在`Message`事件中
```go
func (h *Handler) Message(request iface.IRequest) {

    // 消息
    message := request.GetMessage()
    
    // 来自那个连接的
    connect := request.GetConnect()
    
    // 判断是什么消息类型
    if message.IsText() {
        fmt.Println(connect.Text(message.Bytes()))
    } else {
        fmt.Println(connect.Binary(message.Bytes()))
    }
}
```

## 安装autobahn-testsuite
```bash
docker run -itd  \
    -v "${PWD}/config:/config" 
    -v "${PWD}/reports:/reports" \
    --name fuzzingclient \
    crossbario/autobahn-testsuite bash
```

## 开始测试
```bash
docker exec -it fuzzingclient wstest -m fuzzingclient -s /config/fuzzingclient.json
```

## 测试结果
* 在 `reports/clients/index.html`