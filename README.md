# 介绍
这是一个兼容mysql的哪吒v0分支，虽然支持，但还是不建议使用mysql选项，毕竟就是一个小探针

适用于某些没有持久化的容器，或者说你的数据库性能很好，再或者不折腾就想死可以上，其他时候情况选择关掉mysql

# 特点
- **grpc和http同端口，多路复用实现**
- 优化索引, 不再有记录删除或查询800ms~1600ms的情况出现

配置示例如下

```yaml
debug: false
httpport: 443
language: zh_CN
grpcport: 443 # 若grpcport = httpport时，自动开启多路复用
grpchost: xxxx

dbtype: mysql # postgres / mysql 若使用sqlite可以不填
mysqlconf:
    mysqlPort: 3306
    MysqlHost: test-db-mysql.ns-qrfw0fhx.svc
    MysqlUser: root
    MysqlPwd: 42vb49c4
    MysqlDatabase: nezha
postgresqlconf:
      PGHost: "xxx.postgres.host"
      PGUser: "postgres"
      PGPwd:  "123456"
      PGDatabase: public
      PGPort: 5432
      PrepareStmt: false
      
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

## 无痛迁移
建议选择sqlite-to-mysql

## docker镜像
`docker pull twoonefour1/nezha:v0`
