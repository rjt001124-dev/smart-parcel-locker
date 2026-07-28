// Order status view models. UI actions are derived ONLY from server order
// status; the frontend never fabricates payment or door-open success.

export type OrderStatus =
  | "ORDER_STATUS_UNSPECIFIED"
  | "ORDER_STATUS_PENDING_PAYMENT"
  | "ORDER_STATUS_PAID"
  | "ORDER_STATUS_AWAITING_DEPOSIT"
  | "ORDER_STATUS_IN_STORAGE"
  | "ORDER_STATUS_AWAITING_PICKUP"
  | "ORDER_STATUS_OVERDUE"
  | "ORDER_STATUS_COMPLETED"
  | "ORDER_STATUS_CANCELLED";

export type StatusTone = "success" | "warning" | "danger" | "info" | "muted";

export interface OrderStatusView {
  status: OrderStatus;
  label: string;
  tone: StatusTone;
}

const STATUS_VIEWS: Record<OrderStatus, { label: string; tone: StatusTone }> = {
  ORDER_STATUS_UNSPECIFIED: { label: "未知状态", tone: "muted" },
  ORDER_STATUS_PENDING_PAYMENT: { label: "待支付", tone: "warning" },
  ORDER_STATUS_PAID: { label: "已支付", tone: "success" },
  ORDER_STATUS_AWAITING_DEPOSIT: { label: "待存入", tone: "info" },
  ORDER_STATUS_IN_STORAGE: { label: "寄存中", tone: "info" },
  ORDER_STATUS_AWAITING_PICKUP: { label: "待取件", tone: "info" },
  ORDER_STATUS_OVERDUE: { label: "已逾期", tone: "danger" },
  ORDER_STATUS_COMPLETED: { label: "已完成", tone: "success" },
  ORDER_STATUS_CANCELLED: { label: "已取消", tone: "muted" }
};

export function toOrderStatusView(status: OrderStatus): OrderStatusView {
  const view = STATUS_VIEWS[status] ?? STATUS_VIEWS.ORDER_STATUS_UNSPECIFIED;
  return { status, label: view.label, tone: view.tone };
}

export type OrderAction =
  | "pay"
  | "cancel"
  | "openDepositDoor"
  | "openPickupDoor"
  | "settleOverdue"
  | "contactSupport";

const STATUS_ACTIONS: Record<OrderStatus, OrderAction[]> = {
  ORDER_STATUS_UNSPECIFIED: [],
  ORDER_STATUS_PENDING_PAYMENT: ["pay", "cancel"],
  ORDER_STATUS_PAID: [],
  ORDER_STATUS_AWAITING_DEPOSIT: ["openDepositDoor"],
  ORDER_STATUS_IN_STORAGE: ["openPickupDoor"],
  ORDER_STATUS_AWAITING_PICKUP: ["openPickupDoor"],
  ORDER_STATUS_OVERDUE: ["settleOverdue"],
  ORDER_STATUS_COMPLETED: [],
  ORDER_STATUS_CANCELLED: []
};

/**
 * Actions the UI may offer for a given server-provided status.
 * contactSupport is always available as a fallback channel.
 */
export function availableOrderActions(status: OrderStatus): OrderAction[] {
  return [...(STATUS_ACTIONS[status] ?? []), "contactSupport"];
}

const ACTION_LABELS: Record<OrderAction, string> = {
  pay: "去支付",
  cancel: "取消订单",
  openDepositDoor: "开门存入",
  openPickupDoor: "开门取件",
  settleOverdue: "补缴逾期费",
  contactSupport: "联系客服"
};

export function orderActionLabel(action: OrderAction): string {
  return ACTION_LABELS[action];
}
