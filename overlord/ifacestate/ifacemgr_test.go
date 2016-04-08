// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package ifacestate_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/interfaces"
	"github.com/ubuntu-core/snappy/overlord/ifacestate"
	"github.com/ubuntu-core/snappy/overlord/state"
	"github.com/ubuntu-core/snappy/snap"
)

func TestInterfaceManager(t *testing.T) { TestingT(t) }

type interfaceManagerSuite struct {
	state *state.State
	mgr   *ifacestate.InterfaceManager
}

var _ = Suite(&interfaceManagerSuite{})

func (s *interfaceManagerSuite) SetUpTest(c *C) {
	state := state.New(nil)
	mgr, err := ifacestate.Manager(state)
	c.Assert(err, IsNil)
	s.state = state
	s.mgr = mgr
}

func (s *interfaceManagerSuite) TearDownTest(c *C) {
	s.mgr.Stop()
}

func (s *interfaceManagerSuite) TestSmoke(c *C) {
	s.mgr.Ensure()
	s.mgr.Wait()
}

func (s *interfaceManagerSuite) TestConnectTask(c *C) {
	s.state.Lock()
	defer s.state.Unlock()

	ts, err := ifacestate.Connect(s.state, "consumer", "plug", "producer", "slot")
	c.Assert(err, IsNil)

	task := ts.Tasks()[0]
	c.Assert(task.Kind(), Equals, "connect")
	var plug interfaces.PlugRef
	err = task.Get("plug", &plug)
	c.Assert(err, IsNil)
	c.Assert(plug.Snap, Equals, "consumer")
	c.Assert(plug.Name, Equals, "plug")
	var slot interfaces.SlotRef
	err = task.Get("slot", &slot)
	c.Assert(err, IsNil)
	c.Assert(slot.Snap, Equals, "producer")
	c.Assert(slot.Name, Equals, "slot")
}

func (s *interfaceManagerSuite) TestEnsureProcessesConnectTask(c *C) {
	s.state.Lock()
	defer s.state.Unlock()

	s.addPlugSlotAndInterface(c)
	change := s.state.NewChange("kind", "summary")
	ts, err := ifacestate.Connect(s.state, "consumer", "plug", "producer", "slot")
	c.Assert(err, IsNil)
	change.AddAll(ts)

	s.state.Unlock()
	s.mgr.Ensure()
	s.mgr.Wait()
	s.state.Lock()

	task := change.Tasks()[0]
	c.Check(task.Kind(), Equals, "connect")
	c.Check(task.Status(), Equals, state.DoneStatus)
	c.Check(change.Status(), Equals, state.DoneStatus)
	repo := s.mgr.Repository()
	c.Check(repo.Interfaces(), DeepEquals, &interfaces.Interfaces{
		Slots: []*interfaces.Slot{{
			SlotInfo: &snap.SlotInfo{
				Snap: &snap.Info{SuggestedName: "producer"}, Name: "slot", Interface: "test",
			},
			Connections: []interfaces.PlugRef{{Snap: "consumer", Name: "plug"}},
		}},
		Plugs: []*interfaces.Plug{{
			PlugInfo: &snap.PlugInfo{
				Snap: &snap.Info{SuggestedName: "consumer"}, Name: "plug", Interface: "test",
			},
			Connections: []interfaces.SlotRef{{Snap: "producer", Name: "slot"}},
		}},
	})
}

func (s *interfaceManagerSuite) TestDisconnectTask(c *C) {
	s.state.Lock()
	defer s.state.Unlock()

	ts, err := ifacestate.Disconnect(s.state, "consumer", "plug", "producer", "slot")
	c.Assert(err, IsNil)

	task := ts.Tasks()[0]
	c.Assert(task.Kind(), Equals, "disconnect")
	var plug interfaces.PlugRef
	err = task.Get("plug", &plug)
	c.Assert(err, IsNil)
	c.Assert(plug.Snap, Equals, "consumer")
	c.Assert(plug.Name, Equals, "plug")
	var slot interfaces.SlotRef
	err = task.Get("slot", &slot)
	c.Assert(err, IsNil)
	c.Assert(slot.Snap, Equals, "producer")
	c.Assert(slot.Name, Equals, "slot")
}

func (s *interfaceManagerSuite) TestEnsureProcessesDisconnectTask(c *C) {
	s.state.Lock()
	defer s.state.Unlock()

	s.addPlugSlotAndInterface(c)
	repo := s.mgr.Repository()
	err := repo.Connect("consumer", "plug", "producer", "slot")
	c.Assert(err, IsNil)
	change := s.state.NewChange("kind", "summary")
	ts, err := ifacestate.Disconnect(s.state, "consumer", "plug", "producer", "slot")
	c.Assert(err, IsNil)
	change.AddAll(ts)

	s.state.Unlock()
	s.mgr.Ensure()
	s.mgr.Wait()
	s.state.Lock()

	task := change.Tasks()[0]
	c.Check(task.Kind(), Equals, "disconnect")
	c.Check(task.Status(), Equals, state.DoneStatus)
	c.Check(change.Status(), Equals, state.DoneStatus)
	c.Check(repo.Interfaces(), DeepEquals, &interfaces.Interfaces{
		// NOTE: the connection is gone now.
		Slots: []*interfaces.Slot{{SlotInfo: &snap.SlotInfo{
			Snap: &snap.Info{SuggestedName: "producer"}, Name: "slot", Interface: "test"}}},
		Plugs: []*interfaces.Plug{{PlugInfo: &snap.PlugInfo{
			Snap: &snap.Info{SuggestedName: "consumer"}, Name: "plug", Interface: "test"}}},
	})
}

func (s *interfaceManagerSuite) addPlugSlotAndInterface(c *C) {
	repo := s.mgr.Repository()
	err := repo.AddInterface(&interfaces.TestInterface{InterfaceName: "test"})
	c.Assert(err, IsNil)
	err = repo.AddSlot(&interfaces.Slot{SlotInfo: &snap.SlotInfo{
		Snap: &snap.Info{SuggestedName: "producer"}, Name: "slot", Interface: "test"}})
	c.Assert(err, IsNil)
	err = repo.AddPlug(&interfaces.Plug{PlugInfo: &snap.PlugInfo{
		Snap: &snap.Info{SuggestedName: "consumer"}, Name: "plug", Interface: "test"}})
	c.Assert(err, IsNil)
}