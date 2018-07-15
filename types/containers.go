package types

import (
	"strings"
	"sync"
)

type Containers struct {
	c    map[string]*Container
	m    sync.RWMutex
	once sync.Once
}

func (cs *Containers) Len() int {
	cs.init()

	cs.m.RLock()
	l := len(cs.c)
	cs.m.RUnlock()
	return l
}

func (cs *Containers) Set(containers []Container) {
	cs.init()

	tempContainers := make(map[string]*Container, len(containers)*2)
	for _, c := range containers {
		tempContainers[c.ID] = &c
		if len(c.ID) >= 12 {
			tempContainers[c.ID[:12]] = &c
		}
	}
	cs.m.Lock()
	cs.c = tempContainers
	cs.m.Unlock()
}

func (cs *Containers) Append(c *Container) {
	cs.init()
	if c == nil {
		return
	}

	cs.m.Lock()
	cs.c[c.ID] = c
	if len(c.ID) >= 12 {
		cs.c[c.ID[:12]] = c
	}
	cs.m.Unlock()
}

func (cs *Containers) Find(cID string) *Container {
	cs.init()

	if c, exist := cs.c[cID]; exist {
		return c
	}
	// didn't find this cid
	rC := &Container{}
	l := 0
	for id, info := range cs.c {
		if strings.HasPrefix(id, cID) {
			l++
			rC = info
		}
	}
	if l == 1 {
		return rC
	}

	return nil
}

func (cs *Containers) SetShell(cID, shell string) {
	cs.init()

	if c, exist := cs.c[cID]; exist {
		c.Shell = shell
	}
}

func (cs *Containers) init() {
	cs.once.Do(func() {
		cs.c = map[string]*Container{}
	})
}
