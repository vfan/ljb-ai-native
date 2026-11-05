package eval

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Store 是单文件 JSON 存储：规模小时够用，接口与真实数据库一致，
// 换成 SQLite/Postgres 时只需要替换这一层。
// 结果只追加、不原地修改，保证每次运行可追溯。
type Store struct {
	mu   sync.RWMutex
	path string
	data Data
}

func OpenStore(path string) (*Store, error) {
	store := &Store{path: path}

	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, &store.data); err != nil {
			return nil, err
		}
	case errors.Is(err, os.ErrNotExist):
		Seed(&store.data)
	default:
		return nil, err
	}

	if len(store.data.Prompts) == 0 {
		Seed(&store.data)
	}
	if store.data.Thresholds == (Thresholds{}) {
		store.data.Thresholds = DefaultThresholds
	}
	if err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

// View 返回一份可安全读取的副本。
func (s *Store) View() Data {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyData := s.data
	copyData.Prompts = append([]Prompt(nil), s.data.Prompts...)
	copyData.Datasets = append([]Dataset(nil), s.data.Datasets...)
	copyData.Cases = append([]Case(nil), s.data.Cases...)
	copyData.Runs = append([]Run(nil), s.data.Runs...)
	copyData.Results = append([]Result(nil), s.data.Results...)
	copyData.Gates = append([]GateDecision(nil), s.data.Gates...)
	return copyData
}

func (s *Store) Update(fn func(*Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fn(&s.data); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) AddRun(run Run) error {
	return s.Update(func(data *Data) error {
		data.Runs = append(data.Runs, run)
		return nil
	})
}

func (s *Store) UpdateRun(id string, fn func(*Run)) error {
	return s.Update(func(data *Data) error {
		for index := range data.Runs {
			if data.Runs[index].ID == id {
				fn(&data.Runs[index])
				return nil
			}
		}
		return errors.New("run not found: " + id)
	})
}

func (s *Store) AddResult(result Result) error {
	return s.Update(func(data *Data) error {
		data.Results = append(data.Results, result)
		return nil
	})
}

func (s *Store) SetThresholds(thresholds Thresholds) error {
	return s.Update(func(data *Data) error {
		data.Thresholds = thresholds
		return nil
	})
}

func (s *Store) AddGate(gate GateDecision) error {
	return s.Update(func(data *Data) error {
		data.Gates = append(data.Gates, gate)
		// 只保留最近 20 条发布记录
		if len(data.Gates) > 20 {
			data.Gates = data.Gates[len(data.Gates)-20:]
		}
		return nil
	})
}

func (s *Store) RunResults(runID string) []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Result
	for _, result := range s.data.Results {
		if result.RunID == runID {
			results = append(results, result)
		}
	}
	return results
}
