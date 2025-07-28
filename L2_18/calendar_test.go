package main

import (
	"testing"
	"time"
)

func TestCalendar_CreateEvent(t *testing.T) {
	calendar := NewCalendar()
	
	// Тест успешного создания события
	date := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	event, err := calendar.CreateEvent(1, date, "New Year Party")
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if event.ID != 1 {
		t.Errorf("Expected ID 1, got %d", event.ID)
	}
	
	if event.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", event.UserID)
	}
	
	if event.Title != "New Year Party" {
		t.Errorf("Expected title 'New Year Party', got %s", event.Title)
	}
	
	// Тест с невалидной датой
	_, err = calendar.CreateEvent(1, time.Time{}, "Invalid Event")
	if err != ErrDateInvalid {
		t.Errorf("Expected ErrDateInvalid, got %v", err)
	}
}

func TestCalendar_UpdateEvent(t *testing.T) {
	calendar := NewCalendar()
	
	// Создаем событие
	date := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	event, _ := calendar.CreateEvent(1, date, "Original Title")
	
	// Обновляем событие
	newDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedEvent, err := calendar.UpdateEvent(event.ID, 1, newDate, "Updated Title")
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if updatedEvent.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", updatedEvent.Title)
	}
	
	// Тест обновления несуществующего события
	_, err = calendar.UpdateEvent(999, 1, newDate, "Non-existent")
	if err != ErrEventNotFound {
		t.Errorf("Expected ErrEventNotFound, got %v", err)
	}
}

func TestCalendar_DeleteEvent(t *testing.T) {
	calendar := NewCalendar()
	
	// Создаем событие
	date := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	event, _ := calendar.CreateEvent(1, date, "To Delete")
	
	// Удаляем событие
	err := calendar.DeleteEvent(event.ID, 1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что событие удалено
	events := calendar.GetEventsForDay(1, date)
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
	
	// Тест удаления несуществующего события
	err = calendar.DeleteEvent(999, 1)
	if err != ErrEventNotFound {
		t.Errorf("Expected ErrEventNotFound, got %v", err)
	}
}

func TestCalendar_GetEventsForDay(t *testing.T) {
	calendar := NewCalendar()
	
	date := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	otherDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Создаем события
	calendar.CreateEvent(1, date, "Event 1")
	calendar.CreateEvent(1, date, "Event 2")
	calendar.CreateEvent(1, otherDate, "Other Day Event")
	calendar.CreateEvent(2, date, "Other User Event")
	
	// Получаем события для конкретного дня и пользователя
	events := calendar.GetEventsForDay(1, date)
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestCalendar_GetEventsForWeek(t *testing.T) {
	calendar := NewCalendar()
	
	// Даты в одной неделе 2023 года
	date1 := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC) // Понедельник
	date2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC) // Воскресенье
	otherWeek := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Следующая неделя
	
	calendar.CreateEvent(1, date1, "Week Event 1")
	calendar.CreateEvent(1, date2, "Week Event 2")
	calendar.CreateEvent(1, otherWeek, "Other Week Event")
	
	events := calendar.GetEventsForWeek(1, date1)
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestCalendar_GetEventsForMonth(t *testing.T) {
	calendar := NewCalendar()
	
	date1 := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	otherMonth := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	
	calendar.CreateEvent(1, date1, "Month Event 1")
	calendar.CreateEvent(1, date2, "Month Event 2")
	calendar.CreateEvent(1, otherMonth, "Other Month Event")
	
	events := calendar.GetEventsForMonth(1, date1)
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
	
	// Проверяем сортировку по дате
	if !events[0].Date.Before(events[1].Date) {
		t.Error("Events should be sorted by date")
	}
}

func TestCalendar_Concurrency(t *testing.T) {
	calendar := NewCalendar()
	date := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	
	// Тест на race conditions
	done := make(chan bool, 10)
	
	// Запускаем несколько горутин для создания событий
	for i := 0; i < 10; i++ {
		go func(id int) {
			calendar.CreateEvent(1, date, "Concurrent Event")
			done <- true
		}(i)
	}
	
	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}
	
	events := calendar.GetEventsForDay(1, date)
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}
}
