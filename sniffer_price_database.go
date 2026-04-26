package gofusretrodb

import "fmt"

// GetGameServerByCode retrieves a server by its URL-safe code (e.g. "boune", "allisteria").
// The lookup is case-insensitive.
func (ds *DatabaseService) GetGameServerByCode(code string) (*ServerModel, error) {
	var server ServerModel
	if err := ds.db.Where("LOWER(code) = LOWER(?)", code).First(&server).Error; err != nil {
		return nil, fmt.Errorf("server with code %q not found: %w", code, err)
	}
	return &server, nil
}

// GetAllGameServers returns all servers, active or not.
func (ds *DatabaseService) GetAllGameServers() ([]ServerModel, error) {
	var servers []ServerModel
	if err := ds.db.Order("id").Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to list game servers: %w", err)
	}
	return servers, nil
}
