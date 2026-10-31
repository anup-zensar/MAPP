package performance

import "strings"

// A file holding any of these is refused in full, no row stored (FR-012); journeys use a pseudonym.
func personalDataColumns() []string {
	return []string{"email", "e-mail", "firstname", "first name", "lastname", "last name",
		"fullname", "full name", "phone", "telephone", "mobile", "address", "postcode", "ipaddress", "ip address"}
}

func PersonalDataIn(headings []string) []string {
	var found []string
	for _, heading := range headings {
		clean := strings.ToLower(strings.TrimSpace(heading))
		for _, banned := range personalDataColumns() {
			if clean == banned || strings.Contains(clean, banned) {
				found = append(found, heading)
				break
			}
		}
	}
	return found
}
