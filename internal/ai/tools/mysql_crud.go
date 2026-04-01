package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MysqlCrudInput 描述数据库执行工具的输入参数。
type MysqlCrudInput struct {
	DSN         string `json:"dsn" jsonschema:"description=The Data Source Name for connecting to the MySQL database, including username, password, host, port, and database name"`
	SQL         string `json:"sql" jsonschema:"description=The SQL query to execute against the MySQL database"`
	OperateType string `json:"operate_type" jsonschema:"description=The type of SQL operation to perform: query, insert, update, or delete"`
}

// mysqlExecOutput 描述数据库执行工具的统一输出。
type mysqlExecOutput struct {
	Success      bool             `json:"success"`
	OperateType  string           `json:"operate_type"`
	RowsAffected int64            `json:"rows_affected,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	Message      string           `json:"message,omitempty"`
}

// NewMysqlCrudTool 创建 MySQL CRUD 工具。
func NewMysqlCrudTool() (tool.BaseTool, error) {
	return utils.InferOptionableTool(
		"mysql_crud",
		"Execute SQL queries against the MySQL database and return results in JSON format. Use this tool when you need to query, insert, update or delete data from the database. The results will be formatted as JSON for easy parsing.",
		func(ctx context.Context, input *MysqlCrudInput, opts ...tool.Option) (string, error) {
			if strings.TrimSpace(input.DSN) == "" {
				return "", fmt.Errorf("dsn is required")
			}
			if strings.TrimSpace(input.SQL) == "" {
				return "", fmt.Errorf("sql is required")
			}

			db, err := gorm.Open(mysql.Open(input.DSN), &gorm.Config{})
			if err != nil {
				return "", err
			}

			operateType := strings.ToLower(strings.TrimSpace(input.OperateType))
			switch operateType {
			case "query":
				var rows []map[string]any
				if err := db.WithContext(ctx).Raw(input.SQL).Scan(&rows).Error; err != nil {
					return "", err
				}
				data, err := json.Marshal(mysqlExecOutput{
					Success:     true,
					OperateType: operateType,
					Rows:        rows,
					Message:     "query executed successfully",
				})
				if err != nil {
					return "", err
				}
				return string(data), nil
			case "insert", "update", "delete":
				// 写操作统一通过 Exec 执行，并只回传受影响行数。
				result := db.WithContext(ctx).Exec(input.SQL)
				if result.Error != nil {
					return "", result.Error
				}
				data, err := json.Marshal(mysqlExecOutput{
					Success:      true,
					OperateType:  operateType,
					RowsAffected: result.RowsAffected,
					Message:      "statement executed successfully",
				})
				if err != nil {
					return "", err
				}
				return string(data), nil
			default:
				return "", fmt.Errorf("unsupported operate_type: %s", input.OperateType)
			}
		},
	)
}
