package model

import (
	"fmt"
	"sync"
)

//Activities represents activities
type Activities struct {
	mux        *sync.RWMutex
	activities []*Activity
	Activity   *Activity
}

func (a *Activities) Len() int {
	return len(a.activities)
}

//Push add activity
func (a *Activities) Push(activity *Activity) {
	a.mux.Lock()
	defer a.mux.Unlock()
	a.activities = append(a.activities, activity)
	a.Activity = activity
}

//Pop removes last activity
func (a *Activities) Pop() *Activity {
	a.mux.Lock()
	defer a.mux.Unlock()
	var result *Activity
	if len(a.activities) > 0 {
		result = a.activities[len(a.activities)-1]
		a.activities = a.activities[:len(a.activities)-1]
	}
	if len(a.activities) > 0 {
		a.Activity = a.activities[len(a.activities)-1]
	}
	return result
}

func (a *Activities) Last() *Activity {
	if a.Activity == nil {
		a.Activity = &Activity{
			NeatlyTag: &NeatlyTag{Tag: "main"},
		}
	}
	return a.Activity
}

func (a *Activities) First() *Activity {
	if a.Len() > 0 {
		return a.activities[0]
	}
	return a.Activity
}

//NewActivities creates a new activites
func NewActivities() *Activities {
	return &Activities{
		mux:        &sync.RWMutex{},
		activities: make([]*Activity, 0),
	}
}
