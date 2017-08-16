package main

import (
	"fmt"
	"time"

	check "gopkg.in/check.v1"

	"github.com/Dataman-Cloud/swan/types"
)

func (s *ApiSuite) TestUpdateWeights(c *check.C) {
	// purge

	err := s.purge(time.Second*30, c)
	c.Assert(err, check.IsNil)
	fmt.Println("TestUpdateWeights() purged")

	// create app
	ver := demoVersion().setName("demo").setCount(5).setCPU(0.01).setMem(5).Get()
	id := s.createApp(ver, c)
	err = s.waitApp(id, types.OpStatusNoop, time.Second*30, c)
	c.Assert(err, check.IsNil)
	fmt.Println("TestUpdateWeights() created")

	// verify app
	app := s.inspectApp(id, c)
	c.Assert(app.Name, check.Equals, "demo")
	c.Assert(app.TaskCount, check.Equals, 5)
	c.Assert(app.VersionCount, check.Equals, 1)
	c.Assert(len(app.Version), check.Equals, 1)
	fmt.Println("TestUpdateWeights() verified")

	// do canay update first
	body := &types.CanaryUpdateBody{
		Version:   demoVersion().setName("demo").setCount(5).setCPU(0.01).setMem(10).Get(),
		Instances: 3,
		Value:     0.5,
		OnFailure: "continue",
		Delay:     0.5,
	}
	s.canaryUpdate(id, body, c)
	err = s.waitApp(id, types.OpStatusCanaryUnfinished, time.Second*180, c)
	c.Assert(err, check.IsNil)

	// verify app
	app = s.inspectApp(id, c)
	c.Assert(app.Name, check.Equals, "demo")
	c.Assert(app.VersionCount, check.Equals, 2)
	c.Assert(len(app.Version), check.Equals, 2)
	c.Assert(app.OpStatus, check.Equals, types.OpStatusCanaryUnfinished)

	// verify app tasks
	tasks := s.listAppTasks(id, c)
	c.Assert(len(tasks), check.Equals, 5)

	var n, m int
	for _, task := range tasks {
		if task.Weight == 67 {
			n++
		}

		if task.Weight == 100 {
			m++
		}
	}

	c.Assert(n, check.Equals, 3)
	c.Assert(m, check.Equals, 2)

	// verify app versions
	vers := s.listAppVersions(id, c)
	c.Assert(len(vers), check.Equals, 2)
	c.Assert(vers[0].Mem, check.Equals, float64(10))

	counter := make(map[string]int)
	for _, task := range tasks {
		if v, ok := counter[task.Version]; ok {
			v++
			counter[task.Version] = v
		} else {
			counter[task.Version] = 1
		}
	}

	c.Assert(counter[vers[0].ID], check.Equals, 3)
	c.Assert(counter[vers[1].ID], check.Equals, 2)

	fmt.Println("TestUpdateWeights() canaried")

	s.updateWeights(id, &types.UpdateWeightsBody{
		Value: 0.8,
	}, c)
	err = s.waitApp(id, types.OpStatusCanaryUnfinished, time.Second*180, c)
	c.Assert(err, check.IsNil)
	fmt.Println("TestUpdateWeights() updated")

	// verify app
	app = s.inspectApp(id, c)
	c.Assert(app.Name, check.Equals, "demo")
	c.Assert(app.VersionCount, check.Equals, 2)
	c.Assert(len(app.Version), check.Equals, 2)
	c.Assert(app.OpStatus, check.Equals, types.OpStatusCanaryUnfinished)

	// verify app tasks
	tasks = s.listAppTasks(id, c)
	c.Assert(len(tasks), check.Equals, 5)

	new := 0
	for _, task := range tasks {
		if task.Version == vers[0].ID {
			if task.Weight == 267 {
				new++
			}
		}
	}

	c.Assert(new, check.Equals, 3)

	// switch all traffics
	s.updateWeights(id, &types.UpdateWeightsBody{
		Value: 1.0,
	}, c)
	err = s.waitApp(id, types.OpStatusCanaryUnfinished, time.Second*180, c)
	c.Assert(err, check.IsNil)
	fmt.Println("TestUpdateWeights() updated")

	// verify app
	app = s.inspectApp(id, c)
	c.Assert(app.Name, check.Equals, "demo")
	c.Assert(app.VersionCount, check.Equals, 2)
	c.Assert(len(app.Version), check.Equals, 2)
	c.Assert(app.OpStatus, check.Equals, types.OpStatusCanaryUnfinished)

	// verify app tasks
	tasks = s.listAppTasks(id, c)
	c.Assert(len(tasks), check.Equals, 5)

	new, old := 0, 0
	for _, task := range tasks {
		if task.Version == vers[0].ID {
			if task.Weight == 100 {
				new++
			}
		}

		if task.Weight == 0 {
			old++
		}
	}

	c.Assert(new, check.Equals, 3)
	c.Assert(old, check.Equals, 2)
	// clean up

	err = s.removeApp(id, time.Second*10, c)
	c.Assert(err, check.IsNil)
	fmt.Println("TestCanaryUpdate() removed")
}
