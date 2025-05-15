# 介绍
这是一个兼容mysql的哪吒v0分支
配置示例如下


```yaml
debug: false
httpport: 80
language: zh_CN
grpcport: 443
grpchost: xxxx

UseMysql: true
MysqlPort: 3306
MysqlHost: test-db-mysql.ns-qrfw0fhx.svc
MysqlUser: root
MysqlPwd: 42vb49c4
MysqlDatabase: nezha

tls: true
oauth2:
    type: "github"
    admin: "twoonefour"
    clientid: "xxx"
    clientsecret: "xx"
    endpoint: ""
site:
    brand: "twoonefour的小鸡们"
    cookiename: "nezha-dashboard" 
    theme: "server-status"
```

## 为什么要mysql
等sqlite数据丢了你就老实了


## 无痛迁移
使用sqlite-to-mysql

## docker镜像
`twoonefour1/nezha:v0`
