package model

//Task represents a group of action
type Task struct {
	When        string    //criteria to run this task
	Seq         int       //sequence of the task
	Name        string    //Id of the task
	Description string    //description
	Actions     []*Action //actions
	Init        Variables //variables to initialise state before this taks runs
	Post        Variables //variable to update state after this task completes
	TimeSpentMs int       //optional min required time spent in this task, remaining will force Sleep
}

//HasTagID checks if task has supplied tagIDs
func (t *Task) HasTagID(tagIDs map[string]bool) bool {
	if tagIDs == nil {
		return false
	}
	for _, action := range t.Actions {
		if tagIDs[action.TagID] {
			return true
		}
	}
	return false
}
