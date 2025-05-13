// internal/analyzer/complexity.go
package analyzer

import (
	"regexp"
	"strings"
)

func AnalyzeQueryComplexity(sql string) string {
	sql = strings.ToLower(sql)
	
	joinCount := strings.Count(sql, "join")
	
	hasAggregation := strings.Contains(sql, "group by") || 
		strings.Contains(sql, "count(") || 
		strings.Contains(sql, "sum(") || 
		strings.Contains(sql, "avg(") ||
		strings.Contains(sql, "max(") ||
		strings.Contains(sql, "min(")
	
	hasSubquery := strings.Count(sql, "select") > 1
	
	hasOrdering := strings.Contains(sql, "order by")
	
	hasWindowFunc := strings.Contains(sql, "over (") ||
		strings.Contains(sql, "over(") ||
		strings.Contains(sql, "rank()") ||
		strings.Contains(sql, "row_number()")
	
	conditionComplexity := strings.Count(sql, " and ") + strings.Count(sql, " or ")
	
	hasHaving := strings.Contains(sql, "having ")
	
	hasUnion := strings.Contains(sql, "union ")
	
	hasCTE := strings.Contains(sql, "with ") && (strings.Contains(sql, " as (") || strings.Contains(sql, " as("))
	
	if (joinCount > 2 && (hasAggregation || hasSubquery)) || 
	   hasWindowFunc || 
	   hasUnion || 
	   (hasAggregation && hasHaving) || 
	   hasCTE ||
	   conditionComplexity > 5 {
		return "high"
	} else if (joinCount > 0 && (hasAggregation || hasSubquery)) || 
		  (conditionComplexity > 2) || 
		  (joinCount > 1) {
		return "medium"
	} else if joinCount > 0 || hasAggregation || hasSubquery || hasOrdering {
		return "low-medium"
	} else {
		return "low"
	}
}

func AnalyzeTablesInQuery(sql string) []string {
	sql = strings.ToLower(sql)
	
	tableRegex := regexp.MustCompile(`from\s+([a-z0-9_]+)|join\s+([a-z0-9_]+)`)
	matches := tableRegex.FindAllStringSubmatch(sql, -1)
	
	var tables []string
	seen := make(map[string]bool)
	
	for _, match := range matches {
		var tableName string
		if match[1] != "" {
			tableName = match[1]
		} else {
			tableName = match[2]
		}
		
		if tableName == "" || seen[tableName] {
			continue
		}
		
		seen[tableName] = true
		tables = append(tables, tableName)
	}
	
	return tables
}