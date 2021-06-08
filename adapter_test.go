// Copyright 2017 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redisadapter

import (
	"log"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/util"
)

func testGetPolicy(t *testing.T, e *casbin.Enforcer, res [][]string) {
	t.Helper()
	myRes := e.GetPolicy()
	log.Print("Policy: ", myRes)

	if !util.Array2DEquals(res, myRes) {
		t.Error("Policy: ", myRes, ", supposed to be ", res)
	}
}

func TestAdapter(t *testing.T) {
	// Because the DB is empty at first,
	// so we need to load the policy from the file adapter (.CSV) first.
	e, _ := casbin.NewEnforcer("examples/rbac_model.conf", "examples/rbac_policy.csv")

	//a := NewAdapter("tcp", "127.0.0.1:6379")
	// Use the following if Redis has password like "123"
	//a := NewAdapterWithPassword("tcp", "127.0.0.1:6379", "123")
	a := NewAdapterWithOption(WithAddress("127.0.0.1:6379"), WithPassword("123"))
	t.Run("Read the policies from an empty redis", func(t *testing.T) {
		if err := a.LoadPolicy(e.GetModel()); err != nil {
			t.Error("Should not return an error")
		}
	})

	// This is a trick to save the current policy to the DB.
	// We can't call e.SavePolicy() because the adapter in the enforcer is still the file adapter.
	// The current policy means the policy in the Casbin enforcer (aka in memory).
	a.SavePolicy(e.GetModel())

	// Clear the current policy.
	e.ClearPolicy()
	testGetPolicy(t, e, [][]string{})

	// Load the policy from DB.
	a.LoadPolicy(e.GetModel())
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}, {"bob", "data2", "write"}, {"data2_admin", "data2", "read"}, {"data2_admin", "data2", "write"}})

	// Note: you don't need to look at the above code
	// if you already have a working DB with policy inside.

	// Now the DB has policy, so we can provide a normal use case.
	// Create an adapter and an enforcer.
	// NewEnforcer() will load the policy automatically.
	a = NewAdapterWithOption(WithAddress("127.0.0.1:6379"), WithPassword("123"))
	// Use the following if Redis has password like "123"
	//a := NewAdapterWithPassword("tcp", "127.0.0.1:6379", "123")

	e, _ = casbin.NewEnforcer("examples/rbac_model.conf", a)
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}, {"bob", "data2", "write"}, {"data2_admin", "data2", "read"}, {"data2_admin", "data2", "write"}})

	// Add one policy to DB
	a.AddPolicy("p", "p", []string{"paul", "data2", "read"})
	e.ClearPolicy()
	a.LoadPolicy(e.GetModel())
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}, {"bob", "data2", "write"}, {"data2_admin", "data2", "read"}, {"data2_admin", "data2", "write"}, {"paul", "data2", "read"}})

	// Remove one policy from DB
	a.RemovePolicy("p", "p", []string{"paul", "data2", "read"})
	e.ClearPolicy()
	a.LoadPolicy(e.GetModel())
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}, {"bob", "data2", "write"}, {"data2_admin", "data2", "read"}, {"data2_admin", "data2", "write"}})

	// Add policies to DB
	a.AddPolicies("p", "p", [][]string{
		{"curry", "data1", "write"},
		{"kobe", "data2", "read"},
	})
	e.ClearPolicy()
	a.LoadPolicy(e.GetModel())
	testGetPolicy(t, e, [][]string{
		{"alice", "data1", "read"},
		{"bob", "data2", "write"},
		{"data2_admin", "data2", "read"},
		{"data2_admin", "data2", "write"},
		{"curry", "data1", "write"},
		{"kobe", "data2", "read"},
	})

	// Remove polices from DB
	a.RemovePolicies("p", "p", [][]string{
		{"curry", "data1", "write"},
		{"kobe", "data2", "read"},
	})
	e.ClearPolicy()
	a.LoadPolicy(e.GetModel())
	testGetPolicy(t, e, [][]string{{"alice", "data1", "read"}, {"bob", "data2", "write"}, {"data2_admin", "data2", "read"}, {"data2_admin", "data2", "write"}})
}
