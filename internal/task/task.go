package task

import (
	"strings"
)

type Task struct {
	uid           string
	Name          string
	CommandString string
	Timeout       uint64
	Delay         uint64
	next          []string
	GiveStdout    bool
	ReadStdin     bool
}

func (t *Task) GetUid() string {
	return t.uid
}
func (t *Task) GetNext() []string {
	return t.next
}

func (t *Task) AddNextUid(uid string) {
	t.next = append(t.next, uid)
}

func (t *Task) SetNextUids(uids []string) {
	t.next = uids
}

type ParallelSegment struct {
	uid       string
	Name      string
	startUids []string
	endUids   []string
}

func (t *Task) AddMemberUid(uid string) {
	t.next = append(t.next, uid)
}

func (t *Task) SetMemberUids(uids []string) {
	t.next = uids
}

func (p *ParallelSegment) GetUid() string            { return p.uid }
func (p *ParallelSegment) GetMemberUids() []string   { return p.startUids }
func (p *ParallelSegment) GetEndpointUids() []string { return p.endUids }

func GetTask(uid string, name string, executable string, args []string, timeout uint64,
	delay uint64, nextUids []string, givestdout bool, readstdin bool) *Task {

	toJoin := []string{executable}
	for _, a := range args {
		toJoin = append(toJoin, a)
	}
	cmd := strings.Join(toJoin, " ")

	task := Task{

		uid:           uid,
		Name:          name,
		CommandString: cmd,
		Timeout:       timeout,
		Delay:         delay,
		next:          nextUids,
		GiveStdout:    givestdout,
		ReadStdin:     readstdin,
	}

	return &task
}

func GetParallelSegment(uid string, name string, startUids []string, endUids []string) *ParallelSegment {

	segment := ParallelSegment{
		uid:       uid,
		Name:      name,
		startUids: startUids,
		endUids:   endUids,
	}

	return &segment
}
