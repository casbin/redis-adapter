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
	"encoding/json"
	"errors"
	"runtime"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gomodule/redigo/redis"
)

// CasbinRule is used to determine which policy line to load.
type CasbinRule struct {
	PType string `xorm:"varchar(100) index"`
	V0    string `xorm:"varchar(100) index"`
	V1    string `xorm:"varchar(100) index"`
	V2    string `xorm:"varchar(100) index"`
	V3    string `xorm:"varchar(100) index"`
	V4    string `xorm:"varchar(100) index"`
	V5    string `xorm:"varchar(100) index"`
}

// Adapter represents the Redis adapter for policy storage.
type Adapter struct {
	network  string
	address  string
	key      string
	password string
	conn     redis.Conn
}

// finalizer is the destructor for Adapter.
func finalizer(a *Adapter) {
	a.conn.Close()
}

func newAdapter(network string, address string, key string, password string) *Adapter {
	a := &Adapter{}
	a.network = network
	a.address = address
	a.key = key
	a.password = password

	// Open the DB, create it if not existed.
	a.open()

	// Call the destructor when the object is released.
	runtime.SetFinalizer(a, finalizer)

	return a
}

// NewAdapter is the constructor for Adapter.
func NewAdapter(network string, address string) *Adapter {
	return newAdapter(network, address, "casbin_rules", "")
}

// NewAdapterWithPassword is the constructor for Adapter.
func NewAdapterWithPassword(network string, address string, password string) *Adapter {
	return newAdapter(network, address, "casbin_rules", password)
}

// NewAdapterWithKey is the constructor for Adapter.
func NewAdapterWithKey(network string, address string, key string) *Adapter {
	return newAdapter(network, address, key, "")
}

func (a *Adapter) open() {
	//redis.Dial("tcp", "127.0.0.1:6379")
	if a.password == "" {
		conn, err := redis.Dial(a.network, a.address)
		if err != nil {
			panic(err)
		}

		a.conn = conn
	} else {
		conn, err := redis.Dial(a.network, a.address, redis.DialPassword(a.password))
		if err != nil {
			panic(err)
		}

		a.conn = conn
	}
}

func (a *Adapter) close() {
	a.conn.Close()
}

func (a *Adapter) createTable() {
}

func (a *Adapter) dropTable() {
}

func loadPolicyLine(line CasbinRule, model model.Model) {
	lineText := line.PType
	if line.V0 != "" {
		lineText += ", " + line.V0
	}
	if line.V1 != "" {
		lineText += ", " + line.V1
	}
	if line.V2 != "" {
		lineText += ", " + line.V2
	}
	if line.V3 != "" {
		lineText += ", " + line.V3
	}
	if line.V4 != "" {
		lineText += ", " + line.V4
	}
	if line.V5 != "" {
		lineText += ", " + line.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

// LoadPolicy loads policy from database.
func (a *Adapter) LoadPolicy(model model.Model) error {
	num, err := redis.Int(a.conn.Do("LLEN", a.key))
	if err == redis.ErrNil {
		return nil
	}
	if err != nil {
		return err
	}
	values, err := redis.Values(a.conn.Do("LRANGE", a.key, 0, num))
	if err != nil {
		return err
	}

	var line CasbinRule
	for _, value := range values {
		text, ok := value.([]byte)
		if !ok {
			return errors.New("the type is wrong")
		}
		err = json.Unmarshal(text, &line)
		if err != nil {
			return err
		}
		loadPolicyLine(line, model)
	}

	return nil
}

func savePolicyLine(ptype string, rule []string) CasbinRule {
	line := CasbinRule{}

	line.PType = ptype
	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

// SavePolicy saves policy to database.
func (a *Adapter) SavePolicy(model model.Model) error {
	a.dropTable()
	a.createTable()

	var texts [][]byte

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			text, err := json.Marshal(line)
			if err != nil {
				return err
			}
			texts = append(texts, text)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			text, err := json.Marshal(line)
			if err != nil {
				return err
			}
			texts = append(texts, text)
		}
	}

	_, err := a.conn.Do("RPUSH", redis.Args{}.Add(a.key).AddFlat(texts)...)
	return err
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)
	text, err := json.Marshal(line)
	if err != nil {
		return err
	}
	_, err = a.conn.Do("RPUSH", a.key, text)
	return err
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)
	text, err := json.Marshal(line)
	if err != nil {
		return err
	}
	_, err = a.conn.Do("LREM", a.key, 1, text)
	return err
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}

// AddPolicies adds policy rules to the storage.
func (a *Adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	var texts [][]byte
	for _, rule := range rules {
		line := savePolicyLine(ptype, rule)
		text, err := json.Marshal(line)
		if err != nil {
			return err
		}
		texts = append(texts, text)
	}
	_, err := a.conn.Do("RPUSH", redis.Args{}.Add(a.key).AddFlat(texts)...)
	return err
}

// RemovePolicies removes policy rules from the storage.
func (a *Adapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	for _, rule := range rules {
		line := savePolicyLine(ptype, rule)
		text, err := json.Marshal(line)
		if err != nil {
			return err
		}
		_, err = a.conn.Do("LREM", a.key, 1, text)
		if err != nil {
			return err
		}
	}
	return nil
}
