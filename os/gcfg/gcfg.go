// Copyright 2017 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package gcfg provides reading, caching and managing for configuration.
package gcfg

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogf/gf/container/garray"
	"github.com/gogf/gf/container/gmap"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/internal/cmdenv"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/os/gfsnotify"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/os/gspath"
)

const (
	// DEFAULT_CONFIG_FILE is the default configuration file name.
	DEFAULT_CONFIG_FILE = "config.toml"
)

// Configuration struct.
type Config struct {
	name  *gtype.String       // Default configuration file name.
	paths *garray.StringArray // Searching path array.
	jsons *gmap.StrAnyMap     // The pared JSON objects for configuration files.
	vc    *gtype.Bool         // Whether do violence check in value index searching. It affects the performance when set true(false in default).
}

// New returns a new configuration management object.
// The parameter <file> specifies the default configuration file name for reading.
func New(file ...string) *Config {
	name := DEFAULT_CONFIG_FILE
	if len(file) > 0 {
		name = file[0]
	}
	c := &Config{
		name:  gtype.NewString(name),
		paths: garray.NewStringArray(true),
		jsons: gmap.NewStrAnyMap(true),
		vc:    gtype.NewBool(),
	}
	// Customized dir path from env/cmd.
	if envPath := cmdenv.Get("gf.gcfg.path").String(); envPath != "" {
		if gfile.Exists(envPath) {
			_ = c.SetPath(envPath)
		} else {
			if errorPrint() {
				glog.Errorf("Configuration directory path does not exist: %s", envPath)
			}
		}
	} else {
		// Dir path of working dir.
		_ = c.SetPath(gfile.Pwd())
		// Dir path of binary.
		if selfPath := gfile.SelfDir(); selfPath != "" && gfile.Exists(selfPath) {
			_ = c.AddPath(selfPath)
		}
		// Dir path of main package.
		if mainPath := gfile.MainPkgPath(); mainPath != "" && gfile.Exists(mainPath) {
			_ = c.AddPath(mainPath)
		}
	}
	return c
}

// filePath returns the absolute configuration file path for the given filename by <file>.
func (c *Config) filePath(file ...string) (path string) {
	name := c.name.Val()
	if len(file) > 0 {
		name = file[0]
	}
	path = c.FilePath(name)
	if path == "" {
		buffer := bytes.NewBuffer(nil)
		if c.paths.Len() > 0 {
			buffer.WriteString(fmt.Sprintf("[gcfg] cannot find config file \"%s\" in following paths:", name))
			c.paths.RLockFunc(func(array []string) {
				index := 1
				for _, v := range array {
					buffer.WriteString(fmt.Sprintf("\n%d. %s", index, v))
					index++
					buffer.WriteString(fmt.Sprintf("\n%d. %s", index, v+gfile.Separator+"config"))
					index++
				}
			})
		} else {
			buffer.WriteString(fmt.Sprintf("[gcfg] cannot find config file \"%s\" with no path set/add", name))
		}
		if errorPrint() {
			glog.Error(buffer.String())
		}
	}
	return path
}

// SetPath sets the configuration directory path for file search.
// The parameter <path> can be absolute or relative path,
// but absolute path is strongly recommended.
func (c *Config) SetPath(path string) error {
	// Absolute path.
	realPath := gfile.RealPath(path)
	if realPath == "" {
		// Relative path.
		c.paths.RLockFunc(func(array []string) {
			for _, v := range array {
				if path, _ := gspath.Search(v, path); path != "" {
					realPath = path
					break
				}
			}
		})
	}
	// Path not exist.
	if realPath == "" {
		buffer := bytes.NewBuffer(nil)
		if c.paths.Len() > 0 {
			buffer.WriteString(fmt.Sprintf("[gcfg] SetPath failed: cannot find directory \"%s\" in following paths:", path))
			c.paths.RLockFunc(func(array []string) {
				for k, v := range array {
					buffer.WriteString(fmt.Sprintf("\n%d. %s", k+1, v))
				}
			})
		} else {
			buffer.WriteString(fmt.Sprintf(`[gcfg] SetPath failed: path "%s" does not exist`, path))
		}
		err := errors.New(buffer.String())
		if errorPrint() {
			glog.Error(err)
		}
		return err
	}
	// Should be a directory.
	if !gfile.IsDir(realPath) {
		err := fmt.Errorf(`[gcfg] SetPath failed: path "%s" should be directory type`, path)
		if errorPrint() {
			glog.Error(err)
		}
		return err
	}
	// Repeated path check.
	if c.paths.Search(realPath) != -1 {
		return nil
	}
	c.jsons.Clear()
	c.paths.Clear()
	c.paths.Append(realPath)
	return nil
}

// SetViolenceCheck sets whether to perform hierarchical conflict check.
// This feature needs to be enabled when there is a level symbol in the key name.
// The default is off.
// Turning on this feature is quite expensive,
// and it is not recommended to allow separators in the key names.
// It is best to avoid this on the application side.
func (c *Config) SetViolenceCheck(check bool) {
	c.vc.Set(check)
	c.Clear()
}

// AddPath adds a absolute or relative path to the search paths.
func (c *Config) AddPath(path string) error {
	// Absolute path.
	realPath := gfile.RealPath(path)
	if realPath == "" {
		// Relative path.
		c.paths.RLockFunc(func(array []string) {
			for _, v := range array {
				if path, _ := gspath.Search(v, path); path != "" {
					realPath = path
					break
				}
			}
		})
	}
	if realPath == "" {
		buffer := bytes.NewBuffer(nil)
		if c.paths.Len() > 0 {
			buffer.WriteString(fmt.Sprintf("[gcfg] AddPath failed: cannot find directory \"%s\" in following paths:", path))
			c.paths.RLockFunc(func(array []string) {
				for k, v := range array {
					buffer.WriteString(fmt.Sprintf("\n%d. %s", k+1, v))
				}
			})
		} else {
			buffer.WriteString(fmt.Sprintf(`[gcfg] AddPath failed: path "%s" does not exist`, path))
		}
		err := errors.New(buffer.String())
		if errorPrint() {
			glog.Error(err)
		}
		return err
	}
	if !gfile.IsDir(realPath) {
		err := fmt.Errorf(`[gcfg] AddPath failed: path "%s" should be directory type`, path)
		if errorPrint() {
			glog.Error(err)
		}
		return err
	}
	// Repeated path check.
	if c.paths.Search(realPath) != -1 {
		return nil
	}
	c.paths.Append(realPath)
	//glog.Debug("[gcfg] AddPath:", realPath)
	return nil
}

// GetFilePath is alias of FilePath.
// Deprecated.
func (c *Config) GetFilePath(file ...string) (path string) {
	return c.FilePath(file...)
}

// GetFilePath returns the absolute path of the specified configuration file.
// If <file> is not passed, it returns the configuration file path of the default name.
// If the specified configuration file does not exist,
// an empty string is returned.
func (c *Config) FilePath(file ...string) (path string) {
	name := c.name.Val()
	if len(file) > 0 {
		name = file[0]
	}
	c.paths.RLockFunc(func(array []string) {
		for _, v := range array {
			if path, _ = gspath.Search(v, name); path != "" {
				break
			}
			if path, _ = gspath.Search(v+gfile.Separator+"config", name); path != "" {
				break
			}
		}
	})
	return
}

// SetFileName sets the default configuration file name.
func (c *Config) SetFileName(name string) *Config {
	c.name.Set(name)
	return c
}

// GetFileName returns the default configuration file name.
func (c *Config) GetFileName() string {
	return c.name.Val()
}

// getJson returns a *gjson.Json object for the specified <file> content.
// It would print error if file reading fails.
// If any error occurs, it return nil.
func (c *Config) getJson(file ...string) *gjson.Json {
	name := c.name.Val()
	if len(file) > 0 {
		name = file[0]
	}
	r := c.jsons.GetOrSetFuncLock(name, func() interface{} {
		content := ""
		filePath := ""
		if content = GetContent(name); content == "" {
			filePath = c.filePath(name)
			if filePath == "" {
				return nil
			}
			content = gfile.GetContents(filePath)
		}
		if j, err := gjson.LoadContent(content, true); err == nil {
			j.SetViolenceCheck(c.vc.Val())
			// Add monitor for this configuration file,
			// any changes of this file will refresh its cache in Config object.
			if filePath != "" {
				_, err = gfsnotify.Add(filePath, func(event *gfsnotify.Event) {
					c.jsons.Remove(name)
				})
				if err != nil && errorPrint() {
					glog.Error(err)
				}
			}
			return j
		} else {
			if errorPrint() {
				if filePath != "" {
					glog.Criticalf(`[gcfg] Load config file "%s" failed: %s`, filePath, err.Error())
				} else {
					glog.Criticalf(`[gcfg] Load configuration failed: %s`, err.Error())
				}
			}
		}
		return nil
	})
	if r != nil {
		return r.(*gjson.Json)
	}
	return nil
}
