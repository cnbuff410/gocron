package gocron

import (
	"fmt"
	"testing"
	"time"
)

func task() {
	fmt.Println("I am a running job.")
}

func taskWithParams(a int, b string) {
	fmt.Println(a, b)
}

func assertEqualTime(t *testing.T, actual, expected time.Time) {
	if actual != expected {
		t.Errorf("actual different than expected want: %v -> got: %v", expected, actual)
	}
}

func TestSecond(*testing.T) {
	defaultScheduler.Every(1).Second().Do(task)
	defaultScheduler.Every(1).Second().Do(taskWithParams, 1, "hello")
	stop := defaultScheduler.Start()
	time.Sleep(5 * time.Second)
	close(stop)
	defaultScheduler.Clear()
}

func TestScheduled(t *testing.T) {
	n := NewScheduler()
	n.Every(1).Second().Do(task)
	if !n.Scheduled(task) {
		t.Fatal("Task was scheduled but function couldn't found it")
	}
}

// This is a basic test for the issue described here: https://github.com/jasonlvhit/gocron/issues/23
func TestScheduler_Weekdays(t *testing.T) {
	scheduler := NewScheduler()

	job1 := scheduler.Every(1).Monday().At("23:59")
	job2 := scheduler.Every(1).Wednesday().At("23:59")
	job1.Do(task)
	job2.Do(task)
	t.Logf("job1 scheduled for %s", job1.NextScheduledTime())
	t.Logf("job2 scheduled for %s", job2.NextScheduledTime())
	if job1.NextScheduledTime() == job2.NextScheduledTime() {
		t.Fail()
		t.Logf("Two jobs scheduled at the same time on two different weekdays should never run at the same time.[job1: %s; job2: %s]", job1.NextScheduledTime(), job2.NextScheduledTime())
	}
}

// This ensures that if you schedule a job for today's weekday, but the time is already passed, it will be scheduled for
// next week at the requested time.
func TestScheduler_WeekdaysTodayAfter(t *testing.T) {
	scheduler := NewScheduler()

	now := time.Now()
	timeToSchedule := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-1, 0, 0, time.Local)

	job := callTodaysWeekday(scheduler.Every(1)).At(fmt.Sprintf("%02d:%02d", timeToSchedule.Hour(), timeToSchedule.Minute()))
	job.Do(task)
	t.Logf("job is scheduled for %s", job.NextScheduledTime())
	if job.NextScheduledTime().Weekday() != timeToSchedule.Weekday() {
		t.Fail()
		t.Logf("Job scheduled for current weekday for earlier time, should still be scheduled for current weekday (but next week)")
	}
	nextWeek := time.Date(now.Year(), now.Month(), now.Day()+7, now.Hour(), now.Minute()-1, 0, 0, time.Local)
	if !job.NextScheduledTime().Equal(nextWeek) {
		t.Fail()
		t.Logf("Job should be scheduled for the correct time next week.")
	}
}

// This is to ensure that if you schedule a job for today's weekday, and the time hasn't yet passed, the next run time
// will be scheduled for today.
func TestScheduler_WeekdaysTodayBefore(t *testing.T) {
	scheduler := NewScheduler()

	now := time.Now()
	timeToSchedule := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, time.Local)

	job := callTodaysWeekday(scheduler.Every(1)).At(fmt.Sprintf("%02d:%02d", timeToSchedule.Hour(), timeToSchedule.Minute()))
	job.Do(task)
	t.Logf("job is scheduled for %s", job.NextScheduledTime())
	if !job.NextScheduledTime().Equal(timeToSchedule) {
		t.Fail()
		t.Logf("Job should be run today, at the set time.")
	}
}

func Test_formatTime(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{
			name:     "normal",
			args:     "16:18",
			wantHour: 16,
			wantMin:  18,
			wantErr:  false,
		},
		{
			name:     "normal",
			args:     "6:18",
			wantHour: 6,
			wantMin:  18,
			wantErr:  false,
		},
		{
			name:     "notnumber",
			args:     "e:18",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "outofrange",
			args:     "25:18",
			wantHour: 25,
			wantMin:  18,
			wantErr:  true,
		},
		{
			name:     "wrongformat",
			args:     "19:18:17",
			wantHour: 0,
			wantMin:  0,
			wantErr:  true,
		},
		{
			name:     "wrongminute",
			args:     "19:1e",
			wantHour: 19,
			wantMin:  0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHour, gotMin, err := formatTime(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotHour != tt.wantHour {
				t.Errorf("formatTime() gotHour = %v, want %v", gotHour, tt.wantHour)
			}
			if gotMin != tt.wantMin {
				t.Errorf("formatTime() gotMin = %v, want %v", gotMin, tt.wantMin)
			}
		})
	}
}

func TestTaskAt(t *testing.T) {
	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule to run in next minute
	now := time.Now()
	// Expected start time
	startTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, loc)
	// Expected next start time day after
	startNext := startTime.AddDate(0, 0, 1)

	// Schedule every day At
	startAt := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute()+1)
	dayJob := s.Every(1).Day().At(startAt)

	dayJobDone := make(chan bool, 1)
	// Job running 5 sec
	dayJob.Do(func() {
		t.Log(time.Now(), "job start")
		time.Sleep(2 * time.Second)
		dayJobDone <- true
		t.Log(time.Now(), "job done")
	})

	// Check first run
	nextRun := dayJob.NextScheduledTime()
	assertEqualTime(t, nextRun, startTime)

	sStop := s.Start()      // Start scheduler
	<-dayJobDone            // Wait job done
	close(sStop)            // Stop scheduler
	time.Sleep(time.Second) // wait for scheduler to reschedule job

	// Check next run
	nextRun = dayJob.NextScheduledTime()
	assertEqualTime(t, nextRun, startNext)
}

func TestDaily(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// schedule next run 1 day
	dayJob := s.Every(1).Day()
	dayJob.scheduleNextRun()
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime(t, dayJob.nextRun, exp)

	// schedule next run 2 days
	dayJob = s.Every(2).Days()
	dayJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+2, 0, 0, 0, 0, loc)
	assertEqualTime(t, dayJob.nextRun, exp)

	// Job running longer than next schedule 1day 2 hours
	dayJob = s.Every(1).Day()
	dayJob.lastRun = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+2, 0, 0, 0, loc)
	dayJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime(t, dayJob.nextRun, exp)

	// At() 2 hours before now
	hour := now.Hour() - 2
	minute := now.Minute()
	startAt := fmt.Sprintf("%02d:%02d", hour, minute)
	dayJob = s.Every(1).Day().At(startAt)
	dayJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime(t, dayJob.nextRun, exp)
}

func TestWeekdayAfterToday(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at next week day
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1).Tuesday()
	case time.Tuesday:
		weekJob = s.Every(1).Wednesday()
	case time.Wednesday:
		weekJob = s.Every(1).Thursday()
	case time.Thursday:
		weekJob = s.Every(1).Friday()
	case time.Friday:
		weekJob = s.Every(1).Saturday()
	case time.Saturday:
		weekJob = s.Every(1).Sunday()
	case time.Sunday:
		weekJob = s.Every(1).Monday()
	}

	// First run
	weekJob.scheduleNextRun()
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
	weekJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)
}

func TestWeekdayBeforeToday(t *testing.T) {
	now := time.Now()

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at day before
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1).Sunday()
	case time.Tuesday:
		weekJob = s.Every(1).Monday()
	case time.Wednesday:
		weekJob = s.Every(1).Tuesday()
	case time.Thursday:
		weekJob = s.Every(1).Wednesday()
	case time.Friday:
		weekJob = s.Every(1).Thursday()
	case time.Saturday:
		weekJob = s.Every(1).Friday()
	case time.Sunday:
		weekJob = s.Every(1).Saturday()
	}

	weekJob.scheduleNextRun()
	exp := time.Date(now.Year(), now.Month(), now.Day()+6, 0, 0, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
	weekJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+6, 0, 0, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)
}

func TestWeekdayAt(t *testing.T) {
	now := time.Now()

	hour := now.Hour()
	minute := now.Minute()
	startAt := fmt.Sprintf("%02d:%02d", hour, minute)

	// Create new scheduler to have clean test env
	s := NewScheduler()

	// Schedule job at next week day
	var weekJob *Job
	switch now.Weekday() {
	case time.Monday:
		weekJob = s.Every(1).Tuesday().At(startAt)
	case time.Tuesday:
		weekJob = s.Every(1).Wednesday().At(startAt)
	case time.Wednesday:
		weekJob = s.Every(1).Thursday().At(startAt)
	case time.Thursday:
		weekJob = s.Every(1).Friday().At(startAt)
	case time.Friday:
		weekJob = s.Every(1).Saturday().At(startAt)
	case time.Saturday:
		weekJob = s.Every(1).Sunday().At(startAt)
	case time.Sunday:
		weekJob = s.Every(1).Monday().At(startAt)
	}

	// First run
	weekJob.scheduleNextRun()
	exp := time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)

	// Simulate job run 7 days before
	weekJob.lastRun = weekJob.nextRun.AddDate(0, 0, -7)
	// Next run
	weekJob.scheduleNextRun()
	exp = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, loc)
	assertEqualTime(t, weekJob.nextRun, exp)
}
