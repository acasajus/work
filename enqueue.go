package work

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

// Enqueuer can enqueue jobs.
type Enqueuer struct {
	Namespace string // eg, "myapp-work"
	Pool      *redis.Pool

	queuePrefix string // eg, "myapp-work:jobs:"
	knownJobs   map[string]int64
}

// NewEnqueuer creates a new enqueuer with the specified Redis namespace and Redis pool.
func NewEnqueuer(namespace string, pool *redis.Pool) *Enqueuer {
	return &Enqueuer{
		Namespace:   namespace,
		Pool:        pool,
		queuePrefix: redisKeyJobsPrefix(namespace),
		knownJobs:   make(map[string]int64),
	}
}

// Enqueue will enqueue the specified job name and arguments. The args param can be nil if no args ar needed.
// Example: e.Enqueue("send_email", work.Q{"addr": "test@example.com"})
func (e *Enqueuer) Enqueue(jobName string, args map[string]interface{}) error {
	job := &Job{
		Name:       jobName,
		ID:         makeIdentifier(),
		EnqueuedAt: nowEpochSeconds(),
		Args:       args,
	}

	rawJSON, err := job.serialize()
	if err != nil {
		return err
	}

	conn := e.Pool.Get()
	defer conn.Close()

	if _, err := conn.Do("LPUSH", e.queuePrefix+jobName, rawJSON); err != nil {
		return err
	}

	if err := e.addToKnownJobs(conn, jobName); err != nil {
		return err
	}

	return nil
}

// EnqueueIn enqueues a job in the scheduled job queue for execution in secondsFromNow seconds.
func (e *Enqueuer) EnqueueIn(jobName string, secondsFromNow int64, args map[string]interface{}) error {
	job := &Job{
		Name:       jobName,
		ID:         makeIdentifier(),
		EnqueuedAt: nowEpochSeconds(),
		Args:       args,
	}

	rawJSON, err := job.serialize()
	if err != nil {
		return err
	}

	conn := e.Pool.Get()
	defer conn.Close()

	_, err = conn.Do("ZADD", redisKeyScheduled(e.Namespace), nowEpochSeconds()+secondsFromNow, rawJSON)
	if err != nil {
		return err
	}

	if err := e.addToKnownJobs(conn, jobName); err != nil {
		return err
	}

	return nil
}

func (e *Enqueuer) EnqueueUnique(jobName string, args map[string]interface{}) (bool, error) {
	uniqueKey, err := redisKeyUniqueJob(e.Namespace, jobName, args)
	if err != nil {
		return false, err
	}

	job := &Job{
		Name:       jobName,
		ID:         makeIdentifier(),
		EnqueuedAt: nowEpochSeconds(),
		Args:       args,
		Unique:     true,
	}

	rawJSON, err := job.serialize()
	if err != nil {
		return false, err
	}

	conn := e.Pool.Get()
	defer conn.Close()

	if err := e.addToKnownJobs(conn, jobName); err != nil {
		return false, err
	}

	script := redis.NewScript(2, redisLuaEnqueueUnique)

	scriptArgs := make([]interface{}, 0, 3)
	scriptArgs = append(scriptArgs, e.queuePrefix+jobName) // KEY[1]
	scriptArgs = append(scriptArgs, uniqueKey)             // KEY[2]
	scriptArgs = append(scriptArgs, rawJSON)

	res, err := redis.String(script.Do(conn, scriptArgs...))

	return res == "ok", err
}

func (e *Enqueuer) addToKnownJobs(conn redis.Conn, jobName string) error {
	needSadd := true
	now := time.Now().Unix()
	t, ok := e.knownJobs[jobName]
	if ok {
		if now < t {
			needSadd = false
		}
	}
	if needSadd {
		if _, err := conn.Do("SADD", redisKeyKnownJobs(e.Namespace), jobName); err != nil {
			return err
		}
		e.knownJobs[jobName] = now + 300
	}

	return nil
}
