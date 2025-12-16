package singleton

import (
	"fmt"
	"gorm.io/gorm/schema"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/patrickmn/go-cache"
	"github.com/soheilhy/cmux"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Version = "0.20.13 mysql魔改版"

var (
	Conf  *model.Config
	Cache *cache.Cache
	DB    *gorm.DB
	Loc   *time.Location
)

func InitTimezoneAndCache() {
	var err error
	Loc, err = time.LoadLocation(Conf.Location)
	if err != nil {
		panic(err)
	}

	Cache = cache.New(5*time.Minute, 10*time.Minute)
}

// LoadSingleton 加载子服务并执行
func LoadSingleton() {
	loadNotifications() // 加载通知服务
	loadServers()       // 加载服务器列表
	loadCronTasks()     // 加载定时任务
	loadAPI()
	initNAT()
	initDDNS()
}

// InitConfigFromPath 从给出的文件路径中加载配置
func InitConfigFromPath(path string) {
	Conf = &model.Config{}
	err := Conf.Read(path)
	if err != nil {
		panic(err)
	}
}

func initDB(DB *gorm.DB) {
	var err error
	if Conf.Debug {
		DB = DB.Debug()
	}
	err = DB.AutoMigrate(model.Server{}, model.User{},
		model.Notification{}, model.AlertRule{}, model.Monitor{},
		model.MonitorHistory{}, model.Cron{}, model.Transfer{},
		model.ApiToken{}, model.NAT{}, model.DDNSProfile{})
	if err != nil {
		panic(err)
	}
}

// InitDBFromPath 从给出的文件路径中加载数据库
func InitDBFromPath(path string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{
		CreateBatchSize:                          16,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}
	initDB(DB)
}

func InitDBFromMysql() {
	_cnf := Conf.MysqlConf
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		_cnf.MysqlUser,
		_cnf.MysqlPwd,
		_cnf.MysqlHost,
		_cnf.MysqlPort,
		_cnf.MysqlDatabase)
	DB, err = gorm.Open(mysql.Open(dsn),
		&gorm.Config{
			CreateBatchSize: 200,
			NamingStrategy: schema.NamingStrategy{
				TablePrefix: "nezha.",
			},
		})
	if err != nil {
		panic(err)
	}
	initDB(DB)
}

func InitDBFromPostgres() {
	_cnf := Conf.PostGresqlConf
	var err error
	// 組合 Postgres DSN
	// 假設你複用現有的 Mysql 變數，或者你在 Config 裡加了新的變數
	var dsn string
	if _cnf.PGdsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d",
			_cnf.PGHost,     // 或者 Conf.PGHost
			_cnf.PGUser,     // 或者 Conf.PGUser
			_cnf.PGPwd,      // 或者 Conf.PGPwd
			_cnf.PGDatabase, // 或者 Conf.PGDatabase
			_cnf.PGPort,     // 或者 Conf.PGPort
		)
	} else {
		dsn = _cnf.PGdsn
	}
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		CreateBatchSize: 200,
		PrepareStmt:     _cnf.PrepareStmt,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "nezha.",
		},
	})
	if err != nil {
		panic(err)
	}
	DB.Exec("CREATE SCHEMA IF NOT EXISTS nezha")
	initDB(DB)
}

// RecordTransferHourlyUsage 对流量记录进行打点
func RecordTransferHourlyUsage() {
	ServerLock.Lock()
	defer ServerLock.Unlock()
	now := time.Now()
	nowTrimSeconds := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var txs []model.Transfer
	for id, server := range ServerList {
		tx := model.Transfer{
			ServerID: id,
			In:       utils.Uint64SubInt64(server.State.NetInTransfer, server.PrevTransferInSnapshot),
			Out:      utils.Uint64SubInt64(server.State.NetOutTransfer, server.PrevTransferOutSnapshot),
		}
		if tx.In == 0 && tx.Out == 0 {
			continue
		}
		server.PrevTransferInSnapshot = int64(server.State.NetInTransfer)
		server.PrevTransferOutSnapshot = int64(server.State.NetOutTransfer)
		tx.CreatedAt = nowTrimSeconds
		txs = append(txs, tx)
	}
	if len(txs) == 0 {
		return
	}
	err := DB.Create(txs).Error
	if err != nil {
		log.Println("NEZHA>> Cron 流量统计入库", len(txs), err)
		return
	}
	log.Println("NEZHA>> Cron 流量统计入库", len(txs))
}

// CleanMonitorHistory 清理无效或过时的 监控记录 和 流量记录
func CleanMonitorHistory() {
	// 计算可清理流量记录的时长
	var allServerKeep time.Time
	specialServerKeep := make(map[uint64]time.Time)
	var specialServerIDs []uint64
	var alerts []model.AlertRule
	DB.Find(&alerts)
	for _, alert := range alerts {
		for _, rule := range alert.Rules {
			// 是不是流量记录规则
			if !rule.IsTransferDurationRule() {
				continue
			}
			dataCouldRemoveBefore := rule.GetTransferDurationStart().UTC()
			// 判断规则影响的机器范围
			if rule.Cover == model.RuleCoverAll {
				// 更新全局可以清理的数据点
				if allServerKeep.IsZero() || allServerKeep.After(dataCouldRemoveBefore) {
					allServerKeep = dataCouldRemoveBefore
				}
			} else {
				// 更新特定机器可以清理数据点
				for id := range rule.Ignore {
					if specialServerKeep[id].IsZero() || specialServerKeep[id].After(dataCouldRemoveBefore) {
						specialServerKeep[id] = dataCouldRemoveBefore
						specialServerIDs = append(specialServerIDs, id)
					}
				}
			}
		}
	}
	tx := DB.Begin()
	defer tx.Rollback()
	if tx.Error != nil {
		log.Println("NEZHA>> 删除监控数据遇到错误：", tx.Error)
		return
	}
	// 清理已被删除的服务器的监控记录与流量记录
	var monitorsIds []uint64
	DB.Model(&model.Monitor{}).Select("id").Pluck("id", &monitorsIds)
	DB.Unscoped().Delete(&model.MonitorHistory{}, "created_at < ? OR monitor_id NOT IN (?)", time.Now().AddDate(0, 0, -30), monitorsIds)
	// 由于网络监控记录的数据较多，并且前端仅使用了 1 天的数据
	// 考虑到 sqlite 数据量问题，仅保留一天数据，
	// server_id = 0 的数据会用于/service页面的可用性展示
	DB.Unscoped().Delete(&model.MonitorHistory{}, "(created_at < ? AND server_id != 0) OR monitor_id NOT IN (?)", time.Now().AddDate(0, 0, -1), monitorsIds)
	DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (SELECT id FROM servers)")
	for id, couldRemove := range specialServerKeep {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id = ? AND created_at < ?", id, couldRemove)
	}
	if allServerKeep.IsZero() {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?)", specialServerIDs)
	} else {
		DB.Unscoped().Delete(&model.Transfer{}, "server_id NOT IN (?) AND created_at < ?", specialServerIDs, allServerKeep)
	}
	if err := tx.Commit().Error; err != nil {
		log.Println("NEZHA>> 删除监控数据遇到错误：", tx.Error)
	}
}

// IPDesensitize 根据设置选择是否对IP进行打码处理 返回处理后的IP(关闭打码则返回原IP)
func IPDesensitize(ip string) string {
	if Conf.EnablePlainIPInNotification {
		return ip
	}
	return utils.IPDesensitize(ip)
}

func InitCmux() (cmux.CMux, net.Listener, net.Listener, error) {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(int(Conf.GRPCPort)))
	if err != nil {
		log.Fatal(err)
	}

	mu := cmux.New(l)
	grpcL := mu.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	// 其它全部交给 HTTP（既可 HTTP/1.1 也可 HTTP/2）
	httpL := mu.Match(cmux.Any())

	return mu, grpcL, httpL, nil
}
