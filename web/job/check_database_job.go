package job

import (
	"x-ui/web/service"
	"x-ui/database"
	"x-ui/database/model"
	"os"
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
	syncdata := make([]*model.SyncData, 0)
	err := db.Where("`node` = ?",os.LookupEnv("X_UI_NODE_CODE")).Find(&syncdata).Error
	if err == nil {
		//return logger.Warning("Check needUpdate failed")
	
		for _, syncd := range syncdata {
			if syncd.Synced == false {
				db.Model(model.SyncData{}).Where("`node` = ?",os.LookupEnv("X_UI_NODE_CODE")).Update("synced", true)
				j.xrayService.SetToNeedRestart()
			}
		}
	}
}
