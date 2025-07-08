package task

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
