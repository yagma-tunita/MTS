// Package excel 提供 Excel 文件的读取和写入工具。
//
// 基于 github.com/xuri/excelize/v2 库实现，支持 Office Open XML（.xlsx）格式。
// 功能限制：
//   - 读取时只处理第一个 sheet（适用于单 sheet 导入模板）。
//   - 写入时创建一个 sheet（Sheet1），所有列宽固定 15。
//   - 不支持样式、合并单元格、图片、公式等高级功能。
//
// 适用场景：
//   - 批量导入：港口、船舶、航线等基础数据的 Excel 模板上传。
//   - 批量导出：订单、港口等数据导出为 xlsx 下载。
//
// 不适用场景：
//   - 需要复杂格式（颜色、边框、图表）的报表导出。
//     （这种情况建议直接生成 CSV 或调用专业的报表工具）
package excel

import (
	"fmt"
	"mime/multipart"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// ReadSheet 从上传的 Excel 文件中读取第一个 sheet 的全部内容。
//
// 参数：
//   - file: 通过 c.Request.FormFile("file") 获取的上传文件句柄。
//   - fileSize: 文件大小（字节），来自 multipart.FileHeader.Size。
//     当前未实际使用，仅预留接口签名一致性。
//
// 返回值：
//   - [][]string: 所有行数据。第一行是表头，后续行是数据。
//     每行是一个字符串切片，列数可能不固定（空单元格不会出现在切片中）。
//   - error: 文件无法打开、解析失败或行数不足 2 行时返回错误。
//
// 要求：
//   - 文件必须至少有 2 行：1 行表头 + 1 行数据。
//   - 只有第一个 sheet 被读取。
//   - 空单元格不会出现在返回的行的切片中，因此调用方
//     在按索引取值时需检查 len(row) 是否足够。
//
// 使用示例（handler 层）：
//
//	file, header, err := c.Request.FormFile("file")
//	rows, err := excel.ReadSheet(file, header.Size)
//	// rows[0] 是表头，rows[1:] 是数据
//	for _, row := range rows[1:] {
//	    name := row[0]  // 注意：如果第 0 列为空，row 中可能没有第 0 个元素
//	}
//
// 设计决策：返回 [][]string 而非 []map[string]string：
//   使用列索引比列名更灵活，因为不同的导入模板列顺序可能不同。
//   调用方自己维护表头到索引的映射关系。
func ReadSheet(file multipart.File, fileSize int64) ([][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("excel file must have at least one header row and one data row")
	}
	return rows, nil
}

// WriteSheet 根据表头和行数据生成一个 Excel 文件（.xlsx）的字节流。
//
// 参数：
//   - headers: 表头字符串数组，将写入第 1 行。
//   - data: 数据行，每个元素对应一行字符串数组。
//     从第 2 行开始写入，行序与 data 索引一致。
//
// 返回值：
//   - []byte: 可直接写回 HTTP 响应的 Excel 文件字节流。
//   - error: 写入缓冲区失败时返回错误。
//
// 格式特点：
//   - 创建一个名为 "Sheet1" 的工作表。
//   - 所有列宽统一设为 15 个字符宽度（非自动适应，而是固定值）。
//   - 所有单元格内容以字符串形式写入。
//   - 不设置任何单元格格式（字体、颜色、边框等）。
//
// 使用示例（handler 层）：
//
//	bytes, err := excel.WriteSheet(headers, data)
//	c.Header("Content-Disposition", "attachment; filename=ports.xlsx")
//	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", bytes)
//
// 设计决策：列宽固定为 15 而非自动计算最大宽度：
//   自动计算需要遍历所有行求每列的最大字符数，当数据量大时
//   会影响性能。固定 15 在实际使用中可读性尚可，用户拿到文件
//   后在 Excel 中也可以手动调整列宽。
func WriteSheet(headers []string, data [][]string) ([]byte, error) {
	f := excelize.NewFile()
	sheetName := "Sheet1"
	// 写入表头行（第 1 行），列号从 1 开始
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	// 逐行写入数据（从第 2 行开始）
	for rowIdx, row := range data {
		for colIdx, cellVal := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, cellVal)
		}
	}
	// 设置列宽为 15
	for i := 1; i <= len(headers); i++ {
		colName, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheetName, colName, colName, 15)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseFloat 将字符串解析为 float64。解析失败时不报错，返回 0。
// 用于 Excel 导入时将单元格的字符串值转为数值类型。
// 为什么要静默处理错误：Excel 单元格可能包含空字符串或非数字文本，
// 调用方希望在这些情况下得到 0 而不是中断整体导入流程。
func ParseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ParseInt 将字符串解析为 int64。解析失败时不报错，返回 0。
// 使用场景同 ParseFloat，用于 Excel 导入时的字符串转整数。
func ParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
