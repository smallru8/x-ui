package database

import (
	"fmt"
	//"gorm.io/driver/mysql"
	"io/fs"
	"os"
	"path"
	"x-ui/config"
	"x-ui/database/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func initSyncData() error {
	err := db.AutoMigrate(&model.SyncData{})
	if err != nil {
		return err
	}
	node, ok := os.LookupEnv("X_UI_NODE_CODE")
	var count int64
	if !ok {
		node = "0"
		os.Setenv("X_UI_NODE_CODE", "0")
	}
	err = db.Model(&model.SyncData{}).Where("node = ?", node).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		syncdata := &model.SyncData{
			Node:   node,
			Synced: true,
		}
		return db.Create(syncdata).Error
	}
	return nil
}

func initUser() error {
	err := db.AutoMigrate(&model.User{})
	if err != nil {
		return err
	}
	var count int64
	err = db.Model(&model.User{}).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		user := &model.User{
			Username: "admin",
			Password: "admin",
		}
		return db.Create(user).Error
	}
	return nil
}

func initUserTraffic() error {
	return db.AutoMigrate(&model.UserTraffic{})
}

func initInbound() error {
	return db.AutoMigrate(&model.Inbound{})
}

func initSetting() error {
	return db.AutoMigrate(&model.Setting{})
}

func initDomain() error {
	return db.AutoMigrate(&model.Domain{})
}

func InitDB(dbPath string) error {
	dir := path.Dir(dbPath)
	err := os.MkdirAll(dir, fs.ModeDir)
	if err != nil {
		return err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}

	ip, ok0 := os.LookupEnv("X_UI_MYSQL_IP")
	username, ok1 := os.LookupEnv("X_UI_MYSQL_USER")
	passwd, ok2 := os.LookupEnv("X_UI_MYSQL_PASSWD")
	dbname, ok3 := os.LookupEnv("X_UI_MYSQL_DB")
	port, ok4 := os.LookupEnv("X_UI_MYSQL_PORT")

	//SQLite
	if !ok0 || !ok1 || !ok2 || !ok3 || !ok4 {
		db, err = gorm.Open(sqlite.Open(dbPath), c)
	} else { //MySQL
		//mysql dsn
		//dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", username, passwd, ip, port, dbname)
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Taipei", ip, username, passwd, dbname, port)
		db, err = gorm.Open(postgres.Open(dsn), c)
	}

	if err != nil {
		return err
	}

	err = initUser()
	if err != nil {
		return err
	}
	err = initUserTraffic()
	if err != nil {
		return err
	}
	err = initInbound()
	if err != nil {
		return err
	}
	err = initSetting()
	if err != nil {
		return err
	}
	err = initSyncData()
	if err != nil {
		return err
	}
	err = initDomain()
	if err != nil {
		return err
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
