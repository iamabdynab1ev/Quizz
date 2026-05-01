package telegram

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"request-system/internal/entities"
	tgapi "request-system/pkg/telegram"

	"go.uber.org/zap"
)

type executorSelectionScope struct {
	filter    map[string]interface{}
	title     string
	emptyText string
}

func (c *TelegramController) buildExecutorScopeForOrder(order *entities.Order) executorSelectionScope {
	scope := executorSelectionScope{
		filter:    make(map[string]interface{}),
		title:     "👤 *Сотрудники по заявке:*",
		emptyText: "По структуре заявки сотрудники не найдены\\.\n\nВведите ФИО сотрудника для поиска:",
	}

	switch {
	case order.DepartmentID != nil:
		scope.filter["department_id"] = *order.DepartmentID
		scope.title = "👤 *Сотрудники департамента заявки:*"
		scope.emptyText = "В департаменте заявки сотрудники не найдены\\.\n\nВведите ФИО сотрудника для поиска:"
	case order.OtdelID != nil:
		scope.filter["otdel_id"] = *order.OtdelID
		if order.BranchID != nil {
			scope.filter["branch_id"] = *order.BranchID
		}
		scope.title = "👤 *Сотрудники отдела заявки:*"
		scope.emptyText = "В отделе заявки сотрудники не найдены\\.\n\nВведите ФИО сотрудника для поиска:"
	case order.BranchID != nil:
		scope.filter["branch_id"] = *order.BranchID
		scope.title = "👤 *Сотрудники филиала заявки:*"
		scope.emptyText = "В филиале заявки сотрудники не найдены\\.\n\nВведите ФИО сотрудника для поиска:"
	case order.OfficeID != nil:
		scope.filter["office_id"] = *order.OfficeID
		scope.title = "👤 *Сотрудники офиса заявки:*"
		scope.emptyText = "В офисе заявки сотрудники не найдены\\.\n\nВведите ФИО сотрудника для поиска:"
	}

	return scope
}

func (c *TelegramController) buildOrderCardText(
	ctx context.Context,
	order *entities.Order,
	status *entities.Status,
	creator *entities.User,
	executor *entities.User,
) string {
	var (
		orderTypeName     string
		priorityName      string
		departmentName    string
		otdelName         string
		branchName        string
		officeName        string
		equipmentName     string
		equipmentTypeName string
		equipmentAddress  string
		lastComment       string
	)

	var wg sync.WaitGroup
	run := func(enabled bool, fn func()) {
		if !enabled {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(order.OrderTypeID != nil, func() {
		orderTypeName = c.lookupOrderTypeName(ctx, order.OrderTypeID)
	})
	run(order.PriorityID != nil, func() {
		priorityName = c.lookupPriorityName(ctx, order.PriorityID)
	})
	run(order.DepartmentID != nil, func() {
		departmentName = c.lookupDepartmentName(ctx, order.DepartmentID)
	})
	run(order.OtdelID != nil, func() {
		otdelName = c.lookupOtdelName(ctx, order.OtdelID)
	})
	run(order.BranchID != nil, func() {
		branchName = c.lookupBranchName(ctx, order.BranchID)
	})
	run(order.OfficeID != nil, func() {
		officeName = c.lookupOfficeName(ctx, order.OfficeID)
	})
	run(order.EquipmentID != nil, func() {
		equipmentName, equipmentTypeName, equipmentAddress = c.lookupEquipmentDetails(ctx, order.EquipmentID)
	})
	run(true, func() {
		lastComment = c.lookupLastOrderComment(ctx, order.ID)
	})
	wg.Wait()

	address := ""
	if order.Address != nil {
		address = strings.TrimSpace(*order.Address)
	}
	if address == "" {
		address = equipmentAddress
	}

	statusName := ""
	if status != nil {
		statusName = status.Name
	}

	var text strings.Builder
	text.WriteString("📋 *Заявка №")
	text.WriteString(tgapi.EscapeTextForMarkdownV2(strconv.FormatUint(order.ID, 10)))
	text.WriteString("*\n\n")

	text.WriteString("📝 *Название:* ")
	text.WriteString(formatTelegramOptional(order.Name))
	text.WriteString("\n")

	text.WriteString(getStatusEmoji(status))
	text.WriteString(" *Статус:* ")
	text.WriteString(formatTelegramOptional(statusName))
	text.WriteString("\n")

	appendTelegramLineIfPresent(&text, "🏷️ *Тип заявки:* ", orderTypeName)
	appendTelegramLineIfPresent(&text, "🔥 *Приоритет:* ", priorityName)
	appendTelegramLineIfPresent(&text, "🏢 *Департамент:* ", departmentName)
	appendTelegramLineIfPresent(&text, "👥 *Отдел:* ", otdelName)
	appendTelegramLineIfPresent(&text, "🏬 *Филиал:* ", branchName)
	appendTelegramLineIfPresent(&text, "🏠 *Офис:* ", officeName)
	appendTelegramLineIfPresent(&text, "📍 *Адрес:* ", address)
	appendTelegramLineIfPresent(&text, "💻 *Оборудование:* ", equipmentName)
	appendTelegramLineIfPresent(&text, "🧩 *Тип оборудования:* ", equipmentTypeName)

	if creator != nil {
		appendTelegramLineIfPresent(&text, "👤 *Создатель:* ", creator.Fio)
	}

	if executor != nil {
		appendTelegramLineIfPresent(&text, "👨‍💼 *Исполнитель:* ", executor.Fio)
	}

	text.WriteString("🕒 *Создана:* ")
	text.WriteString(formatTelegramDate(order.CreatedAt, c.loc))
	text.WriteString("\n")

	text.WriteString("🕒 *Обновлена:* ")
	text.WriteString(formatTelegramDate(order.UpdatedAt, c.loc))
	text.WriteString("\n")

	if order.CompletedAt != nil {
		text.WriteString("✅ *Завершена:* ")
		text.WriteString(formatTelegramDate(*order.CompletedAt, c.loc))
		text.WriteString("\n")
	}

	text.WriteString("⏰ *Срок:* ")
	if order.Duration == nil {
		text.WriteString("_не задан_")
	} else {
		durationStr := formatTelegramDate(*order.Duration, c.loc)
		if order.Duration.Before(time.Now().In(c.loc)) {
			text.WriteString("~")
			text.WriteString(durationStr)
			text.WriteString("~ ⚠️ _просрочено_")
		} else {
			text.WriteString(durationStr)
		}
	}
	text.WriteString("\n")

	if lastComment != "" {
		text.WriteString("\n💬 *Последний комментарий:*\n_")
		text.WriteString(tgapi.EscapeTextForMarkdownV2(lastComment))
		text.WriteString("_\n")
	}

	return text.String()
}

func (c *TelegramController) lookupOrderTypeName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	orderType, err := c.orderTypeRepo.FindByID(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: order type lookup failed", zap.Uint64("order_type_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(orderType.Name)
}

func (c *TelegramController) lookupPriorityName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	priority, err := c.priorityRepo.FindByID(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: priority lookup failed", zap.Uint64("priority_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(priority.Name)
}

func (c *TelegramController) lookupDepartmentName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	department, err := c.departmentRepo.FindDepartment(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: department lookup failed", zap.Uint64("department_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(department.Name)
}

func (c *TelegramController) lookupOtdelName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	otdel, err := c.otdelRepo.FindOtdel(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: otdel lookup failed", zap.Uint64("otdel_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(otdel.Name)
}

func (c *TelegramController) lookupBranchName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	branch, err := c.branchRepo.FindBranch(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: branch lookup failed", zap.Uint64("branch_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(branch.Name)
}

func (c *TelegramController) lookupOfficeName(ctx context.Context, id *uint64) string {
	if id == nil {
		return ""
	}

	office, err := c.officeRepo.FindOffice(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: office lookup failed", zap.Uint64("office_id", *id), zap.Error(err))
		return ""
	}

	return strings.TrimSpace(office.Name)
}

func (c *TelegramController) lookupEquipmentDetails(ctx context.Context, id *uint64) (string, string, string) {
	if id == nil {
		return "", "", ""
	}

	equipment, err := c.equipmentRepo.FindEquipment(ctx, *id)
	if err != nil {
		c.logger.Debug("telegram order card: equipment lookup failed", zap.Uint64("equipment_id", *id), zap.Error(err))
		return "", "", ""
	}

	typeName := ""
	if equipment.EquipmentType != nil {
		typeName = strings.TrimSpace(equipment.EquipmentType.Name)
	}

	return strings.TrimSpace(equipment.Name), typeName, strings.TrimSpace(equipment.Address)
}

func (c *TelegramController) lookupLastOrderComment(ctx context.Context, orderID uint64) string {
	history, err := c.orderHistoryRepo.FindByOrderID(ctx, orderID, 10, 0)
	if err != nil {
		c.logger.Debug("telegram order card: history lookup failed", zap.Uint64("order_id", orderID), zap.Error(err))
		return ""
	}

	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Comment.Valid {
			comment := strings.TrimSpace(history[i].Comment.String)
			if comment != "" {
				return comment
			}
		}
	}

	return ""
}

func formatTelegramOptional(value string) string {
	if strings.TrimSpace(value) == "" {
		return "_не указано_"
	}

	return tgapi.EscapeTextForMarkdownV2(strings.TrimSpace(value))
}

func formatTelegramDate(value time.Time, loc *time.Location) string {
	return tgapi.EscapeTextForMarkdownV2(value.In(loc).Format("02.01.2006 15:04"))
}

func appendTelegramLineIfPresent(builder *strings.Builder, label string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	builder.WriteString(label)
	builder.WriteString(tgapi.EscapeTextForMarkdownV2(strings.TrimSpace(value)))
	builder.WriteString("\n")
}
