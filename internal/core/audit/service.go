package audit

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) LogActivity(userID *uint, action, details, ip, userAgent string) error {
	return nil // Audit logging disabled temporarily
	/*
		log := &AuditLog{
			UserID:    userID,
			Action:    action,
			Details:   details,
			IPAddress: ip,
			UserAgent: userAgent,
		}
		return s.repo.CreateAuditLog(log)
	*/
}
