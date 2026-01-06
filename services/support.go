package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"sharifbot/database"
)

type SupportService struct {
	db *gorm.DB
}

func NewSupportService(db *gorm.DB) *SupportService {
	return &SupportService{db: db}
}

// FindAvailableSupport finds an available support agent
func (s *SupportService) FindAvailableSupport() (*database.User, error) {
	var supportAgents []database.User

	// Find online support agents with least active tickets
	err := s.db.Where("is_support = ? AND is_online = ?", true, true).
		Order("(SELECT COUNT(*) FROM support_messages WHERE support_id = users.id AND is_resolved = false)").
		First(&supportAgents).Error

	if err != nil || len(supportAgents) == 0 {
		// Try to find any support agent (even offline)
		err = s.db.Where("is_support = ?", true).
			Order("(SELECT COUNT(*) FROM support_messages WHERE support_id = users.id AND is_resolved = false)").
			First(&supportAgents).Error

		if err != nil || len(supportAgents) == 0 {
			return nil, errors.New("هیچ پشتیبان در دسترس نیست")
		}
	}

	return &supportAgents[0], nil
}

// GetSupportTickets gets tickets for a support agent
func (s *SupportService) GetSupportTickets(supportID uint, resolved bool, page, limit int) ([]database.SupportMessage, int64, error) {
	var tickets []database.SupportMessage
	var total int64

	query := s.db.Model(&database.SupportMessage{}).
		Preload("User").
		Where("support_id = ?", supportID)

	if !resolved {
		query = query.Where("is_resolved = ?", false)
	}

	query.Count(&total)
	err := query.Order("created_at DESC").
		Limit(limit).Offset((page - 1) * limit).
		Find(&tickets).Error

	return tickets, total, err
}

// GetUserTickets gets tickets for a user
func (s *SupportService) GetUserTickets(userID uint, page, limit int) ([]database.SupportMessage, int64, error) {
	var tickets []database.SupportMessage
	var total int64

	query := s.db.Model(&database.SupportMessage{}).
		Preload("Support").
		Where("user_id = ?", userID)

	query.Count(&total)
	err := query.Order("created_at DESC").
		Limit(limit).Offset((page - 1) * limit).
		Find(&tickets).Error

	return tickets, total, err
}

// CreateTicket creates a new support ticket
func (s *SupportService) CreateTicket(userID uint, message string) (*database.SupportMessage, error) {
	// Find available support
	support, err := s.FindAvailableSupport()
	if err != nil {
		return nil, err
	}

	ticket := &database.SupportMessage{
		UserID:     userID,
		SupportID:  &support.ID,
		Message:    message,
		SenderType: "user",
		IsResolved: false,
		CreatedAt:  time.Now(),
	}

	err = s.db.Create(ticket).Error
	return ticket, err
}

// AddMessage adds a message to a ticket
func (s *SupportService) AddMessage(ticketID uint, senderType string, message string, supportID uint) error {
	// Verify ticket exists and get its details
	var ticket database.SupportMessage
	if err := s.db.First(&ticket, ticketID).Error; err != nil {
		return fmt.Errorf("تیکت یافت نشد: %v", err)
	}

	supportMessage := &database.SupportMessage{
		UserID:     ticket.UserID,
		SupportID:  &supportID,
		Message:    message,
		SenderType: senderType,
		IsResolved: ticket.IsResolved,
		CreatedAt:  time.Now(),
	}

	return s.db.Create(supportMessage).Error
}

// ResolveTicket resolves a ticket
func (s *SupportService) ResolveTicket(ticketID uint) error {
	return s.db.Model(&database.SupportMessage{}).
		Where("id = ?", ticketID).
		Update("is_resolved", true).Error
}

// GetSupportStats gets statistics for a support agent
func (s *SupportService) GetSupportStats(supportID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count open tickets
	var openTickets int64
	s.db.Model(&database.SupportMessage{}).
		Where("support_id = ? AND is_resolved = ?", supportID, false).
		Count(&openTickets)
	stats["open_tickets"] = openTickets

	// Count resolved tickets
	var resolvedTickets int64
	s.db.Model(&database.SupportMessage{}).
		Where("support_id = ? AND is_resolved = ?", supportID, true).
		Count(&resolvedTickets)
	stats["resolved_tickets"] = resolvedTickets

	// Total tickets
	stats["total_tickets"] = openTickets + resolvedTickets

	// Average response time (simplified)
	var responseTimes []struct {
		ResponseTime float64
	}

	s.db.Raw(`
		SELECT AVG(JULIANDAY(sm2.created_at) - JULIANDAY(sm1.created_at)) * 86400 as response_time
		FROM support_messages sm1
		JOIN support_messages sm2 ON sm1.id = sm2.id - 1
		WHERE sm1.sender_type = 'user' 
		AND sm2.sender_type = 'support'
		AND sm2.support_id = ?
	`, supportID).Scan(&responseTimes)

	if len(responseTimes) > 0 && responseTimes[0].ResponseTime > 0 {
		stats["avg_response_time"] = fmt.Sprintf("%.0f ثانیه", responseTimes[0].ResponseTime)
	} else {
		stats["avg_response_time"] = "ندارد"
	}

	// Recent activity (last 10 tickets)
	var recentTickets []database.SupportMessage
	s.db.Preload("User").
		Where("support_id = ?", supportID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentTickets)
	stats["recent_tickets"] = recentTickets

	// Today's activity
	today := time.Now().Truncate(24 * time.Hour)
	var todayTickets int64
	s.db.Model(&database.SupportMessage{}).
		Where("support_id = ? AND created_at >= ?", supportID, today).
		Count(&todayTickets)
	stats["today_tickets"] = todayTickets

	return stats, nil
}

// GetAllSupportStats gets statistics for all support agents
func (s *SupportService) GetAllSupportStats() ([]map[string]interface{}, error) {
	var supportAgents []database.User
	if err := s.db.Where("is_support = ?", true).Find(&supportAgents).Error; err != nil {
		return nil, err
	}

	var allStats []map[string]interface{}
	for _, agent := range supportAgents {
		stats, err := s.GetSupportStats(agent.ID)
		if err != nil {
			continue
		}
		stats["agent"] = map[string]interface{}{
			"id":        agent.ID,
			"full_name": agent.FullName,
			"is_online": agent.IsOnline,
		}
		allStats = append(allStats, stats)
	}

	return allStats, nil
}

// UpdateSupportStatus updates support agent's online status
func (s *SupportService) UpdateSupportStatus(supportID uint, isOnline bool) error {
	return s.db.Model(&database.User{}).
		Where("id = ?", supportID).
		Update("is_online", isOnline).Error
}

// GetUnresolvedTickets gets all unresolved tickets
func (s *SupportService) GetUnresolvedTickets(page, limit int) ([]database.SupportMessage, int64, error) {
	var tickets []database.SupportMessage
	var total int64

	query := s.db.Model(&database.SupportMessage{}).
		Preload("User").
		Preload("Support").
		Where("is_resolved = ?", false)

	query.Count(&total)
	err := query.Order("created_at ASC"). // Oldest first
						Limit(limit).Offset((page - 1) * limit).
						Find(&tickets).Error

	return tickets, total, err
}

// AssignTicketToSupport assigns a ticket to a specific support agent
func (s *SupportService) AssignTicketToSupport(ticketID, supportID uint) error {
	return s.db.Model(&database.SupportMessage{}).
		Where("id = ?", ticketID).
		Update("support_id", supportID).Error
}

// GetTicketMessages gets all messages for a specific ticket
func (s *SupportService) GetTicketMessages(ticketID uint) ([]database.SupportMessage, error) {
	// First get the main ticket to know the user and support
	var mainTicket database.SupportMessage
	if err := s.db.First(&mainTicket, ticketID).Error; err != nil {
		return nil, err
	}

	// Get all messages for this user with this support (including the main ticket)
	var messages []database.SupportMessage
	err := s.db.Where("user_id = ? AND support_id = ?", mainTicket.UserID, mainTicket.SupportID).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

// GetSupportPerformance gets performance metrics for support agents
func (s *SupportService) GetSupportPerformance(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Query support performance metrics
	rows, err := s.db.Raw(`
		SELECT 
			u.id,
			u.full_name,
			COUNT(DISTINCT sm.id) as total_tickets,
			SUM(CASE WHEN sm.is_resolved = 1 THEN 1 ELSE 0 END) as resolved_tickets,
			AVG(CASE WHEN sm.is_resolved = 1 THEN 1 ELSE 0 END) * 100 as resolution_rate
		FROM users u
		LEFT JOIN support_messages sm ON u.id = sm.support_id
		WHERE u.is_support = 1
		AND sm.created_at BETWEEN ? AND ?
		GROUP BY u.id
		ORDER BY total_tickets DESC
	`, startDate, endDate).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result struct {
			ID              uint    `json:"id"`
			FullName        string  `json:"full_name"`
			TotalTickets    int     `json:"total_tickets"`
			ResolvedTickets int     `json:"resolved_tickets"`
			ResolutionRate  float64 `json:"resolution_rate"`
		}

		err := rows.Scan(&result.ID, &result.FullName, &result.TotalTickets,
			&result.ResolvedTickets, &result.ResolutionRate)
		if err != nil {
			continue
		}

		stats := map[string]interface{}{
			"id":               result.ID,
			"full_name":        result.FullName,
			"total_tickets":    result.TotalTickets,
			"resolved_tickets": result.ResolvedTickets,
			"resolution_rate":  fmt.Sprintf("%.1f%%", result.ResolutionRate),
		}

		results = append(results, stats)
	}

	return results, nil
}
