package task

import "fmt"

type Task struct {
	uid        string
	Name       string
	Executable string
	Args       []string
	Timeout    uint64
	Delay      uint64
	next       []string
	GiveStdout bool
	ReadStdin  bool
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

func (s *ParallelSegment) AddStartUid(uid string) {
	s.startUids = append(s.startUids, uid)
}

func (s *ParallelSegment) AddEndUid(uid string) {
	s.endUids = append(s.endUids, uid)
}
func (s *ParallelSegment) SetStartUids(uids []string) {
	s.startUids = uids
}
func (s *ParallelSegment) SetEndUids(uids []string) {
	s.endUids = uids
}

func (p *ParallelSegment) GetUid() string          { return p.uid }
func (p *ParallelSegment) GetMemberUids() []string { return p.startUids }
func (p *ParallelSegment) GetEndpointUids() []string {
	fmt.Println("segment enduids:", p.endUids)
	return p.endUids
}

func GetTask(uid string, name string, executable string, args []string, timeout uint64,
	delay uint64, nextUids []string, givestdout bool, readstdin bool) *Task {

	toJoin := []string{executable}
	for _, a := range args {
		toJoin = append(toJoin, a)
	}

	task := Task{

		uid:        uid,
		Name:       name,
		Executable: executable,
		Args:       args,
		Timeout:    timeout,
		Delay:      delay,
		next:       nextUids,
		GiveStdout: givestdout,
		ReadStdin:  readstdin,
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
	fmt.Println(segment)
	return &segment
}
