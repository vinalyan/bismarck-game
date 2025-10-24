package restrictions

// RestrictedHexesForGermanDD содержит гексы, которые немецкие эсминцы не могут пересекать
var RestrictedHexesForGermanDD = []string{
	"Q29", "R28", "S27", "T26",
}

// ConvoyHexes содержит гексы конвоев, в которые немецкие танкеры не могут входить
var ConvoyHexes = []string{
	"H15", "I16", "J17", // Основной маршрут конвоя
	"K18", "L19", "M20", // Продолжение маршрута
	"N21", "O22", "P23", // Дополнительные гексы конвоя
}

// IsRestrictedHexForGermanDD проверяет, является ли гекс ограниченным для немецких эсминцев
func IsRestrictedHexForGermanDD(hex string) bool {
	for _, restrictedHex := range RestrictedHexesForGermanDD {
		if hex == restrictedHex {
			return true
		}
	}
	return false
}

// IsConvoyHex проверяет, является ли гекс гексом конвоя
func IsConvoyHex(hex string) bool {
	for _, convoyHex := range ConvoyHexes {
		if hex == convoyHex {
			return true
		}
	}
	return false
}
