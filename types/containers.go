package types

import (
	"strings"
	"sync"
)

type Containers struct {
	c    map[string]Container
	cs   []Container
	m    sync.RWMutex
	once sync.Once
}

func (cs *Containers) List() []Container {
	cs.init()
	return cs.cs
}

func (cs *Containers) Len() int {
	cs.init()

	return len(cs.c)
}

func (cs *Containers) Set(containers []Container) {
	cs.init()

	mapContainers := make(map[string]Container, len(containers)*2)
	for _, c := range containers {
		mapContainers[c.ID] = c
		if len(c.ID) >= 12 {
			mapContainers[c.ID[:12]] = c
		}
	}
	cs.m.Lock()
	cs.c = mapContainers
	cs.cs = containers
	cs.m.Unlock()
}

func (cs *Containers) Append(c Container) {
	cs.init()
	if c.ID == "" {
		return
	}

	cs.m.Lock()
	cs.c[c.ID] = c
	if len(c.ID) >= 12 {
		cs.c[c.ID[:12]] = c
	}
	cs.cs = append(cs.cs, c)
	cs.m.Unlock()
}

func (cs *Containers) Find(cID string) Container {
	cs.init()

	cs.m.RLock()
	defer cs.m.RUnlock()

	if c, exist := cs.c[cID]; exist {
		return c
	}
	// didn't find this cid
	rC := Container{}
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

	return Container{}
}

func (cs *Containers) SetShell(cID, shell string) {
	cs.init()
	cs.m.Lock()
	defer cs.m.Unlock()

	if c, exist := cs.c[cID]; exist {
		c.Shell = shell
		cs.c[cID] = c
	}
}

func (cs *Containers) init() {
	cs.once.Do(func() {
		cs.c = make(map[string]Container, 100)
	})
}
