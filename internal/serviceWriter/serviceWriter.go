package serviceWriter

type Unit struct {
	Description string
	After       string
}

type Service struct {
	Type            string
	ExecString      string
	RemainAfterExit string
}

type Timer struct {
	OnBootSec       string
	OnUnitActiveSec string
}

type Install struct {
	WantedBy string
}

type SystemdService interface {
	GetAsString() string
}

type ImmediateService struct {
	Unit
	Service
	Install
}

func (s *ImmediateService) GetAsString() string {

}

type ScheduledService struct {
	Unit
	Service
	Timer
	Install
}

func (s *ScheduledService) GetAsString() string {

}

func GetImmediateService() ImmediateService {

}

func GetScheduledService() ScheduledService {

}
