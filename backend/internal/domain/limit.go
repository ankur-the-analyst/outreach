package domain

func DefaultDailyLimit(ageDays int) int {
	if ageDays < 30 {
		return 40
	}
	return 100
}
