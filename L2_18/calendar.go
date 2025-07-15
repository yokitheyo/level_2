package main

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrDateInvalid   = errors.New("invalid date")
)

type Event struct {
	ID     int       `json:"id"`
	UserID int       `json:"user_id"`
	Date   time.Time `json:"date"`
	Title  string    `json:"title"`
}

type Calendar struct {
	mu     sync.RWMutex
	events []Event
	nextID int
}

func NewCalendar() *Calendar {
	return &Calendar{
		events: make([]Event, 0),
		nextID: 1,
	}
}

func (c *Calendar) CreateEvent(userID int, date time.Time, title string) (Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if date.IsZero() {
		return Event{}, ErrDateInvalid
	}

	event := Event{c.nextID, userID, date, title}
	c.nextID++
	c.events = append(c.events, event)

	return event, nil
}

func (c *Calendar) UpdateEvent(id, userID int, date time.Time, title string) (Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if date.IsZero() {
		return Event{}, ErrDateInvalid
	}

	for i, event := range c.events {
		if event.ID == id && event.UserID == userID {
			updatedEvent := Event{id, userID, date, title}
			c.events[i] = updatedEvent

			return updatedEvent, nil
		}
	}
	return Event{}, ErrEventNotFound
}

func (c *Calendar) DeleteEvent(id, userID int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, event := range c.events {
		if event.ID == id && event.UserID == userID {
			c.events = append(c.events[:i], c.events[i+1:]...)
			return nil
		}
	}
	return ErrEventNotFound
}

func (c *Calendar) GetEventsForDay(userID int, date time.Time) []Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Event
	for _, event := range c.events {
		if event.UserID == userID && isSameDay(event.Date, date) {
			result = append(result, event)
		}
	}
	return result
}

func (c *Calendar) GetEventsForWeek(userID int, date time.Time) []Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Event
	year, week := date.ISOWeek()
	for _, event := range c.events {
		if event.UserID == userID {
			eYear, eWeek := event.Date.ISOWeek()
			if eYear == year && eWeek == week {
				result = append(result, event)
			}
		}
	}
	return result
}

func (c *Calendar) GetEventsForMonth(userID int, date time.Time) []Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Event
	year, month := date.Year(), date.Month()
	for _, event := range c.events {
		if event.UserID == userID && event.Date.Year() == year && event.Date.Month() == month {
			result = append(result, event)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result
}

func isSameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
