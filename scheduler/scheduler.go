package scheduler

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
	"github.com/robfig/cron"

	"einheit/boltkit/entity"
	"einheit/boltkit/service"
	"einheit/boltkit/util"
)

var (
	// The job scheduler.
	AppScheduler *Scheduler
)

// Scheduler represents a background job scheduler.
type Scheduler struct {
	Ch   chan string
	Cron *cron.Cron
}

// NewScheduler creates a background task scheduler
func NewScheduler() *Scheduler {
	scheduler := new(Scheduler)
	scheduler.Cron = cron.New()
	scheduler.Ch = make(chan string)
	return scheduler
}

// Send forwards jobs for processing.
func (scheduler *Scheduler) Send(job string) {
	scheduler.Ch <- job
}

// Schedule prioritizes jobs according to the alloted times set.
func (scheduler *Scheduler) Schedule(app *service.Service) {
	// Scheduled to run at 8pm each day.
	scheduler.Cron.AddFunc("0 0 20 * * *", func() { scheduler.Send(util.InviteJob) })
	// Scheduled to run at 9pm each day.
	scheduler.Cron.AddFunc("0 0 21 * * *", func() { scheduler.Send(util.PassResetJob) })

	log.Info("Scheduled recurring jobs.")
}

// Process receives and executes the posted job.
func (scheduler *Scheduler) Process(app *service.Service) {
	for {
		job := <-scheduler.Ch
		switch job {
		case util.InviteJob:
			ExpiredInvites(app)
		case util.InviteJob:
			ExpiredPassReset(app)
		default:
			log.Error("unknown job received: ", job)
		}
	}
}

// ExpiredInvites removes expired invitations from storage.
func ExpiredInvites(app *service.Service) {
	expiredInvites := []entity.Invite{}
	err := app.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.InviteBucket)
		cursor := bucket.Cursor()
		invite := new(entity.Invite)
		now := time.Now().Unix()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, invite)
			if err != nil {
				return err
			}

			// Only load expired or cancelled invites.
			if (now > invite.Expiry && invite.Status != entity.Pending) || invite.Status == entity.Cancelled {
				expiredInvites = append(expiredInvites, *invite)
			}
		}

		return nil
	})

	if err != nil {
		log.Error("expired invites job failed: ", err)
	}

	if len(expiredInvites) > 0 {
		err = app.Bolt.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(util.InviteBucket)
			for _, invite := range expiredInvites {
				err := bucket.Delete([]byte(invite.Uuid))
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Error("expired invites job failed: ", err)
		}
	}
}

// ExpiredPassReset removes expired password resets from storage.
func ExpiredPassReset(app *service.Service) {
	expiredResets := []entity.PassReset{}
	err := app.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.PassResetBucket)
		cursor := bucket.Cursor()
		reset := new(entity.PassReset)
		now := time.Now().Unix()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, reset)
			if err != nil {
				return err
			}

			// Only load expired and unused password resets.
			if now > reset.Expiry && !reset.Used {
				expiredResets = append(expiredResets, *reset)
			}
		}

		return nil
	})

	if err != nil {
		log.Error("expired password resets job failed: ", err)
	}

	if len(expiredResets) > 0 {
		err = app.Bolt.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(util.PassResetBucket)
			for _, reset := range expiredResets {
				err := bucket.Delete([]byte(reset.Uuid))
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Error("expired password resets job failed: ", err)
		}
	}
}
