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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gomodule/redigo/redis"
)

// CasbinRule is used to determine which policy line to load.
type CasbinRule struct {
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

// Adapter represents the Redis adapter for policy storage.
type Adapter struct {
	network    string
	address    string
	key        string
	username   string
	password   string
	conn       redis.Conn
	isFiltered bool
}

// finalizer is the destructor for Adapter.
func finalizer(a *Adapter) {
	a.conn.Close()
}

func newAdapter(network string, address string, key string,
	username string, password string) *Adapter {
	a := &Adapter{}
	a.network = network
	a.address = address
	a.key = key
	a.username = username
	a.password = password

	// Open the DB, create it if not existed.
	a.open()

	// Call the destructor when the object is released.
	runtime.SetFinalizer(a, finalizer)

	return a
}

// NewAdapter is the constructor for Adapter.
func NewAdapter(network string, address string) *Adapter {
	return newAdapter(network, address, "casbin_rules", "", "")
}

func NewAdapterWithUser(network string, address string, username string, password string) *Adapter {
	return newAdapter(network, address, "casbin_rules", username, password)
}

// NewAdapterWithPassword is the constructor for Adapter.
func NewAdapterWithPassword(network string, address string, password string) *Adapter {
	return newAdapter(network, address, "casbin_rules", "", password)
}

// NewAdapterWithKey is the constructor for Adapter.
func NewAdapterWithKey(network string, address string, key string) *Adapter {
	return newAdapter(network, address, key, "", "")
}

type Option func(*Adapter)

func NewAdpaterWithOption(options ...Option) *Adapter {
	a := &Adapter{}
	for _, option := range options {
		option(a)
	}
	// Open the DB, create it if not existed.
	a.open()

	// Call the destructor when the object is released.
	runtime.SetFinalizer(a, finalizer)

	return a
}

func WithAddress(address string) Option {
	return func(a *Adapter) {
		a.address = address
	}
}

func WithUsername(username string) Option {
	return func(a *Adapter) {
		a.username = username
	}
}

func WithPassword(password string) Option {
	return func(a *Adapter) {
		a.password = password
	}
}

func WithNetwork(network string) Option {
	return func(a *Adapter) {
		a.network = network
	}
}
func WithKey(key string) Option {
	return func(a *Adapter) {
		a.key = key
	}
}

func (a *Adapter) open() {
	//redis.Dial("tcp", "127.0.0.1:6379")
	if a.username != "" {
		conn, err := redis.Dial(a.network, a.address, redis.DialUsername(a.username), redis.DialPassword(a.password))
		if err != nil {
			panic(err)
		}

		a.conn = conn
	} else if a.password == "" {
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
	_, _ = a.conn.Do("DEL", a.key)
}

func (c *CasbinRule) toStringPolicy() []string {
	policy := make([]string, 0)
	if c.PType != "" {
		policy = append(policy, c.PType)
	}
	if c.V0 != "" {
		policy = append(policy, c.V0)
	}
	if c.V1 != "" {
		policy = append(policy, c.V1)
	}
	if c.V2 != "" {
		policy = append(policy, c.V2)
	}
	if c.V3 != "" {
		policy = append(policy, c.V3)
	}
	if c.V4 != "" {
		policy = append(policy, c.V4)
	}
	if c.V5 != "" {
		policy = append(policy, c.V5)
	}
	return policy
}

func loadPolicyLine(line CasbinRule, model model.Model) {
	text := line.toStringPolicy()

	persist.LoadPolicyArray(text, model)
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

	a.isFiltered = false
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

//FilteredAdapter

// IsFiltered returns true if the loaded policy has been filtered.
func (a *Adapter) IsFiltered() bool {
	return a.isFiltered
}

type Filter struct {
	PType []string
	V0    []string
	V1    []string
	V2    []string
	V3    []string
	V4    []string
	V5    []string
}

func filterToRegexPattern(filter *Filter) string {
	// example data in redis: {"PType":"p","V0":"data2_admin","V1":"data2","V2":"write","V3":"","V4":"","V5":""}

	var f = [][]string{filter.PType,
		filter.V0, filter.V1, filter.V2,
		filter.V3, filter.V4, filter.V5}

	args := []interface{}{}
	for _, v := range f {
		if len(v) == 0 {
			args = append(args, ".*")
		} else {
			escapedV := make([]string, 0, len(v))
			for _, s := range v {
				escapedV = append(escapedV, regexp.QuoteMeta(s))
			}
			args = append(args, "(?:"+strings.Join(escapedV, "|")+")") // (?:data2_admin|data1_admin)
		}
	}

	// example pattern:
	//^\{"PType":".*","V0":"(?:data2_admin|data1_admin)","V1":".*","V2":".*","V3":".*","V4":".*","V5":".*"\}$
	pattern := fmt.Sprintf(
		`^\{"PType":"%s","V0":"%s","V1":"%s","V2":"%s","V3":"%s","V4":"%s","V5":"%s"\}$`, args...,
	)
	return pattern
}

func escapeLuaPattern(s string) string {
	var buf bytes.Buffer
	for _, char := range s {
		switch char {
		case '.', '%', '-', '+', '*', '?', '^', '$', '(', ')', '[', ']': // magic chars: . % + - * ? [ ( ) ^ $
			buf.WriteRune('%')
		}
		buf.WriteRune(char)
	}
	return buf.String()
}

func filterFieldToLuaPattern(sec string, ptype string, fieldIndex int, fieldValues ...string) string {
	args := []interface{}{ptype}

	idx := fieldIndex + len(fieldValues)
	for i := 0; i < 6; i++ { // v0-v5
		if fieldIndex <= i && idx > i && fieldValues[i-fieldIndex] != "" {
			args = append(args, escapeLuaPattern(fieldValues[i-fieldIndex]))
		} else {
			args = append(args, ".*")
		}
	}

	// example pattern:
	// ^{"PType":"p","V0":"data2_admin","V1":".*","V2":".*","V3":".*","V4":".*","V5":".*"}$
	pattern := fmt.Sprintf(
		`^{"PType":"%s","V0":"%s","V1":"%s","V2":"%s","V3":"%s","V4":"%s","V5":"%s"}$`, args...,
	)
	return pattern
}

func (a *Adapter) loadFilteredPolicy(model model.Model, filter *Filter) error {
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

	re := regexp.MustCompile(filterToRegexPattern(filter))

	var line CasbinRule
	for _, value := range values {
		text, ok := value.([]byte)
		if !ok {
			return errors.New("the type is wrong")
		}

		if !re.Match(text) {
			continue
		}

		err = json.Unmarshal(text, &line)
		if err != nil {
			return err
		}
		loadPolicyLine(line, model)
	}
	return nil
}

// LoadFilteredPolicy loads only policy rules that match the filter.
func (a *Adapter) LoadFilteredPolicy(model model.Model, filter interface{}) error {
	if filter == nil {
		return a.LoadPolicy(model)
	}

	var err error
	switch f := filter.(type) {
	case *Filter:
		err = a.loadFilteredPolicy(model, f)
	case Filter:
		err = a.loadFilteredPolicy(model, &f)
	default:
		err = fmt.Errorf("invalid filter type")
	}

	if err != nil {
		return err
	}
	a.isFiltered = true
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {

	pattern := filterFieldToLuaPattern(sec, ptype, fieldIndex, fieldValues...)

	var getScript = redis.NewScript(1, `
		local key = KEYS[1]
		local pattern = ARGV[1]
		
		local r = redis.call('lrange', key, 0, -1)
		for i=1, #r do 
			if  string.find(r[i], pattern) then
				redis.call('lset', key, i-1, '__CASBIN_DELETED__')
			end
		end
		redis.call('lrem', key, 0, '__CASBIN_DELETED__')
		return 
	`)
	_, err := getScript.Do(a.conn, a.key, pattern)
	return err
}

// UpdatableAdapter

// UpdatePolicy updates a new policy rule to DB.
func (a *Adapter) UpdatePolicy(sec string, ptype string, oldRule, newPolicy []string) error {
	oldLine := savePolicyLine(ptype, oldRule)
	textOld, err := json.Marshal(oldLine)
	if err != nil {
		return err
	}
	newLine := savePolicyLine(ptype, newPolicy)
	textNew, err := json.Marshal(newLine)
	if err != nil {
		return err
	}

	var getScript = redis.NewScript(1, `
		local key = KEYS[1]
		local old = ARGV[1]
		local newRule = ARGV[2]
	
		local r = redis.call('lrange', key, 0, -1)
		for i=1,#r do
			if r[i] == old then
				redis.call('lset', key, i-1, newRule)
				return true
			end
		end
		return false
	`)
	_, err = getScript.Do(a.conn, a.key, textOld, textNew)
	return err
}

func (a *Adapter) UpdatePolicies(sec string, ptype string, oldRules, newRules [][]string) error {

	if len(oldRules) != len(newRules) {
		return errors.New("oldRules and newRules should have the same length")
	}

	oldPolicies := make([]string, 0, len(oldRules))
	newPolicies := make([]string, 0, len(newRules))
	for _, oldRule := range oldRules {
		textOld, err := json.Marshal(savePolicyLine(ptype, oldRule))
		if err != nil {
			return err
		}
		oldPolicies = append(oldPolicies, string(textOld))
	}
	for _, newRule := range newRules {
		textNew, err := json.Marshal(savePolicyLine(ptype, newRule))
		if err != nil {
			return err
		}
		newPolicies = append(newPolicies, string(textNew))
	}

	// Initialize a package-level variable with a script.
	var getScript = redis.NewScript(1, `
		local key = KEYS[1]
		local len = #ARGV/2
		
		local map = {}
		for i = 1, len, 1 do
			map[ARGV[i]] = ARGV[i + len] -- map[oldRule] = newRule
		end
		
		local r = redis.call('lrange', key, 0, -1)
		for i=1,#r do
			if map[r[i]] ~= nil then
				redis.call('lset', key, i-1, map[r[i]])
				-- return true
			end
		end
		
		return false
	`)
	args := redis.Args{}.Add(a.key).AddFlat(oldPolicies).AddFlat(newPolicies)
	_, err := getScript.Do(a.conn, args...)
	return err
}

func (a *Adapter) UpdateFilteredPolicies(sec string, ptype string, newPolicies [][]string, fieldIndex int, fieldValues ...string) ([][]string, error) {
	// UpdateFilteredPolicies deletes old rules and adds new rules.

	oldP := make([]string, 0)
	newP := make([]string, 0, len(newPolicies))
	for _, newRule := range newPolicies {
		textNew, err := json.Marshal(savePolicyLine(ptype, newRule))
		if err != nil {
			return nil, err
		}
		newP = append(newP, string(textNew))
	}

	pattern := filterFieldToLuaPattern(sec, ptype, fieldIndex, fieldValues...)

	// Initialize a package-level variable with a script.
	var getScript = redis.NewScript(1, `
		local key = KEYS[1]
		local pattern = ARGV[1]
		
		local ret = {}
		local r = redis.call('lrange', key, 0, -1)
		for i=1, #r do 
			if  string.find(r[i], pattern) then
        		table.insert(ret, r[i])
				redis.call('lset', key, i-1, '__CASBIN_DELETED__')
			end
		end
		redis.call('lrem', key, 0, '__CASBIN_DELETED__')
		
		local r = redis.call('lrange', key, 0, -1)
		for i=2,#r do
			redis.call('rpush', key, ARGV[i])
		end
		
		return ret
	`)
	args := redis.Args{}.Add(a.key).Add(pattern).AddFlat(newP)
	//r, err := getScript.Do(a.conn, args...)
	//reply, err := redis.Values(r, err)
	reply, err := redis.Values(getScript.Do(a.conn, args...))
	if err != nil {
		return nil, err
	}

	if err = redis.ScanSlice(reply, &oldP); err != nil {
		return nil, err
	}

	ret := make([][]string, 0, len(oldP))
	for _, oldRule := range oldP {
		var line CasbinRule
		if err := json.Unmarshal([]byte(oldRule), &line); err != nil {
			return nil, err
		}

		ret = append(ret, line.toStringPolicy())
	}

	return ret, nil
}
