package job

import (
	"x-ui/web/service"
	"x-ui/logger"
	"x-ui/database"
	"x-ui/database/model"
)

type CheckDatabaseJob struct {
	xrayService    service.XrayService
	inboundService service.InboundService
}

func NewCheckDatabaseJob() *CheckDatabaseJob {
	return new(CheckDatabaseJob)
}

func (j *CheckDatabaseJob) Run() {
	db := database.GetDB()
	settings := make([]*model.Setting, 0)
	err := db.Where("key = ?","needUpdate").Find(&settings).Error
	if err != nil {
		return logger.Warning("disable invalid inbounds err:", err)
	}
	for _, setting := range settings {
		if setting.Value == "true" {
			db.Model(model.Setting{}).Where("key = ?", "needUpdate").Update("value", "false")
			j.xrayService.SetToNeedRestart()
		}
	}
}
