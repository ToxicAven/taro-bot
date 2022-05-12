package plugins

import (
	"github.com/5HT2/taro-bot/bot"
	"log"
)

// ClearJobs will go through bot.Jobs and handle the de-registration of them
func ClearJobs() {
	for _, job := range bot.Jobs {
		_ = job.Scheduler.RemoveByTag(job.Tag)
	}

	bot.Jobs = make([]bot.JobInfo, 0)
}

// RegisterJobs will go through bot.Jobs and handle the re-registration of them
func RegisterJobs() {
	for _, job := range bot.Jobs {
		// Run job if it doesn't have checking enabled, or if the condition is true
		if !job.CheckCondition || job.Condition {
			if rJob, err := job.Scheduler.Tag(job.Tag).Do(job.Fn); err != nil {
				log.Printf("failed to register job (%s): %v\n", job.Tag, err)
			} else {
				log.Printf("registered job: %v\n", rJob)
			}
		}
	}
}

// RegisterAll will register all bot features, and then load plugins
func RegisterAll(dir, pluginList string) {
	bot.Mutex.Lock()
	defer bot.Mutex.Unlock()

	// This is done to clear the existing plugins that have already been registered, if this is called after the bot
	// has already been initialized. This allows reloading plugins at runtime.
	bot.Commands = make([]bot.CommandInfo, 0)
	bot.Responses = make([]bot.ResponseInfo, 0)

	// We want to do this before registering plugins
	ClearJobs()

	// This registers the plugins we have downloaded
	// This does not build new plugins for us, which instead has to be done separately
	Load(dir, pluginList)

	// This registers the new jobs that plugins have scheduled
	RegisterJobs()
}
