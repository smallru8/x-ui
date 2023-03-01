package service

import (
	"fmt"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/util/common"
	"x-ui/xray"
	"encoding/json"
	"gorm.io/gorm"
	"strings"
)

type InboundService struct {
}

/////////////////////////////////
type Client_data struct {
    Email string `json:email`
    Level int64 `json:level`
    Id string `json:"id"`
    AlterId int64 `json:alterId`
}

type Setting_data struct {
    Clients []*Client_data `json:"clients"`
    DisableInsecureEncryption bool `json:disableInsecureEncryption`
}

func remove(slice []*Client_data, s int) []*Client_data {
    return append(slice[:s], slice[s+1:]...)
}
/////////////////////////////////

func (s *InboundService) GetInbounds(userId int) ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Where("user_id = ?", userId).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) GetAllInbounds() ([]*model.Inbound, error) {
	db := database.GetDB()
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) checkPortExist(port int, ignoreId int) (bool, error) {
	db := database.GetDB()
	db = db.Model(model.Inbound{}).Where("port = ?", port)
	if ignoreId > 0 {
		db = db.Where("id != ?", ignoreId)
	}
	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *InboundService) AddInbound(inbound *model.Inbound) error {
	exist, err := s.checkPortExist(inbound.Port, 0)
	if err != nil {
		return err
	}
	if exist {
		return common.NewError("端口已存在:", inbound.Port)
	}
	db := database.GetDB()
	return db.Save(inbound).Error
}

func (s *InboundService) AddInbounds(inbounds []*model.Inbound) error {
	for _, inbound := range inbounds {
		exist, err := s.checkPortExist(inbound.Port, 0)
		if err != nil {
			return err
		}
		if exist {
			return common.NewError("端口已存在:", inbound.Port)
		}
	}

	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	for _, inbound := range inbounds {
		err = tx.Save(inbound).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *InboundService) DelInbound(id int) error {
	db := database.GetDB()
	return db.Delete(model.Inbound{}, id).Error
}

func (s *InboundService) GetInbound(id int) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).First(inbound, id).Error
	if err != nil {
		return nil, err
	}
	return inbound, nil
}

func (s *InboundService) UpdateInbound(inbound *model.Inbound) error {
	exist, err := s.checkPortExist(inbound.Port, inbound.Id)
	if err != nil {
		return err
	}
	if exist {
		return common.NewError("端口已存在:", inbound.Port)
	}

	oldInbound, err := s.GetInbound(inbound.Id)
	if err != nil {
		return err
	}
	oldInbound.Up = inbound.Up
	oldInbound.Down = inbound.Down
	oldInbound.Total = inbound.Total
	oldInbound.Remark = inbound.Remark
	oldInbound.Enable = inbound.Enable
	oldInbound.ExpiryTime = inbound.ExpiryTime
	oldInbound.Listen = inbound.Listen
	oldInbound.Port = inbound.Port
	oldInbound.Protocol = inbound.Protocol
	oldInbound.Settings = inbound.Settings
	oldInbound.StreamSettings = inbound.StreamSettings
	oldInbound.Sniffing = inbound.Sniffing
	oldInbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)

	db := database.GetDB()
	return db.Save(oldInbound).Error
}

func (s *InboundService) AddTraffic(traffics []*xray.Traffic) (err error) {
	if len(traffics) == 0 {
		return nil
	}
	db := database.GetDB()
	txUser := db.Model(model.UserTraffic{}).Begin()
	db = db.Model(model.Inbound{})
	tx := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			txUser.Rollback()
		} else {
			tx.Commit()
			txUser.Commit()
		}
	}()
	for _, traffic := range traffics {
		if traffic.IsUser {
			//err = txUser.Where("tag = ?", traffic.Tag).
			//	UpdateColumn("up", gorm.Expr("up + ?", traffic.Up)).
			//	UpdateColumn("down", gorm.Expr("down + ?", traffic.Down)).
			//	Error
			err = txUser.Where("tag = ?", traffic.Tag).Updates(map[string]interface{}{"up" : gorm.Expr("up + ?", traffic.Up) , "down" : gorm.Expr("down + ?", traffic.Down)}).Error
			if err != nil {
				return
			}
		} else if traffic.IsInbound {
			//err = tx.Where("tag = ?", traffic.Tag).
			//	UpdateColumn("up", gorm.Expr("up + ?", traffic.Up)).
			//	UpdateColumn("down", gorm.Expr("down + ?", traffic.Down)).
			//	Error
			err = tx.Where("tag = ?", traffic.Tag).Updates(map[string]interface{}{"up" : gorm.Expr("up + ?", traffic.Up) , "down" : gorm.Expr("down + ?", traffic.Down)}).Error
			if err != nil {
				return
			}
		}
	}
	return
}

//只有Master執行異動資料庫function，Slave只統計流量
func (s *InboundService) DisableInvalidInbounds() (int64, error) {
	db := database.GetDB()
	now := time.Now().Unix() * 1000
	result := db.Model(model.Inbound{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	err := result.Error
	count := result.RowsAffected
	
	//要通知slave重啟
	return count, err
}

//只有Master執行異動資料庫function，Slave只統計流量
func (s *InboundService) AdjustUsers() (count int64, err error) {
	db := database.GetDB()
	now := time.Now().Unix() * 1000
	
	txuser := db.Model(model.UserTraffic{}).Begin()
	txinb := db.Model(model.Inbound{}).Begin()
	
	defer func() {
		if err != nil {
			txuser.Rollback()
			txinb.Rollback()
		} else {
			txuser.Commit()
			txinb.Commit()
		}
	}()
	
	users := make([]*model.UserTraffic, 0)
	//waitenable 表示帳號等待重新載入, 所以先從 inbound 移出,待下一步移入
	err = db.Where("(((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?) or (enable = ? and wait_enable = ?)",now,true,true,true).Find(&users).Error
	count = 0
	if err == nil && len(users) > 0 {//有需要調整的使用者
		count = int64(len(users))
		inbs := make([]*model.Inbound, 0)
		err = db.Find(&inbs).Error
		if err == nil {
			for _, user := range users {
				for j := 0 ; j<len(inbs) ; j = j+1 {
					dataJson := inbs[j].Settings
					var jsonct Setting_data
					_ = json.Unmarshal([]byte(dataJson), &jsonct)
					
					for i := len(jsonct.Clients)-1 ; i >= 0 ; i = i-1 {
						if jsonct.Clients[i].Email == user.Tag && jsonct.Clients[i].Id == user.Uuid {
							jsonct.Clients = remove(jsonct.Clients,i)
						}
					}
					rawBytes, err := json.Marshal(jsonct)
					if err == nil {//存回去
						inbs[j].Settings = string(rawBytes)
					}
				}
			}
			
			for _, inb := range inbs {//存回DB
				err = txinb.Where("`id` = ?",inb.Id).Update("settings", inb.Settings).Error
				if err != nil {
					return count, err
				}
			}
			err = txuser.Where("(((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?) or (enable = ? and wait_enable = ?)",now,true,true,true).Update("enable", false).Error
			if err != nil {
				return count, err
			}
		}
	}
	
	//需要被重新載入的 user
	users = make([]*model.UserTraffic, 0)
	//取所有要重新載入的 user
	err = db.Where("enable = ? and wait_enable = ?",false,true).Find(&users).Error
	if err == nil && len(users) > 0 {
		count = int64(len(users))
		//取所有 inbound
		inbs := make([]*model.Inbound, 0)
		err = db.Find(&inbs).Error
		for _, user := range users {
			countries := strings.Split(user.Country," ")
			for _, country := range countries {
				for i := 0 ; i < len(inbs) ; i = i+1 {
					if inbs[i].Remark == country {//將 user 加入
						var jsonct Setting_data
						_ = json.Unmarshal([]byte(inbs[i].Settings), &jsonct)
						////////////////////////////建構 json 格式資料
						user_data := &Client_data{
							Email: user.Tag,
							Level: 0,
							Id: user.Uuid,
							AlterId: 10,
						}
						////////////////////////////
						jsonct.Clients = append(jsonct.Clients,user_data)
						rawBytes, err := json.Marshal(jsonct)
						if err == nil {//存回去
							inbs[i].Settings = string(rawBytes)
						}
					}
				}
			}
		}
		for _, inb := range inbs {//存回DB
			err = txinb.Where("`id` = ?",inb.Id).Update("settings", inb.Settings).Error
			if err != nil {
				return count, err
			}
		}
		err = txuser.Where("enable = ? and wait_enable = ?",false,true).Updates(map[string]interface{}{"wait_enable": false, "enable": true}).Error
		if err != nil {
			return count, err
		}
	}
	
	//要通知slave重啟
	return count, err
}
