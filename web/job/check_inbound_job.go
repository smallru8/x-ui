package job

import (
	"x-ui/logger"
	"x-ui/web/service"
	"x-ui/database"
	"x-ui/database/model"
)

type CheckInboundJob struct {
	xrayService    service.XrayService
	inboundService service.InboundService
}

func NewCheckInboundJob() *CheckInboundJob {
	return new(CheckInboundJob)
}

//Master 執行就好
func (j *CheckInboundJob) Run() {
	count, err := j.inboundService.DisableInvalidInbounds()
	count2, err2 := j.inboundService.DisableInvalidUsers()
	if err != nil {
		logger.Warning("disable invalid inbounds err:", err)
	} else if err2 != nil {
		logger.Warning("disable invalid user err:", err2)
	} else if count > 0 || count2 > 0 {
		logger.Debugf("disabled %v inbounds", count)
		//通知其他主機有更動須重啟
		db := database.GetDB()
		db.Model(model.SyncData{}).Where("node != ?",os.LookupEnv("X_UI_NODE_CODE")).Update("Synced",false)
		
		j.xrayService.SetToNeedRestart()
	}
}
