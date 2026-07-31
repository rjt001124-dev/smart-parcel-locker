import type { RequestFn } from "./http";

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

export interface FeeSnapshotDto {
  rent_fee_fen: number;
  deposit_fen: number;
  discount_fen: number;
  total_fen: number;
}

export interface OrderDto {
  id: string;
  order_no: string;
  site_id: string;
  site_name: string;
  cell_id: string;
  cell_no: string;
  size: string;
  status: OrderStatus;
  duration_minutes: number;
  fee_snapshot: FeeSnapshotDto;
  overdue_fee_fen: number;
  created_at: string;
  paid_at: string;
  deposited_at: string;
  expires_at: string;
  completed_at: string;
}

/** protojson emits int64 as string; accept both and coerce to integer. */
function toFen(value: string | number | undefined): number {
  if (value === undefined || value === "") return 0;
  const parsed = typeof value === "number" ? value : Number.parseInt(value, 10);
  if (!Number.isInteger(parsed)) {
    throw new Error(`invalid integer fen amount: ${String(value)}`);
  }
  return parsed;
}

interface RawFeeSnapshotDto {
  rent_fee_fen?: string | number;
  rentFeeFen?: string | number;
  deposit_fen?: string | number;
  depositFen?: string | number;
  discount_fen?: string | number;
  discountFen?: string | number;
  total_fen?: string | number;
  totalFen?: string | number;
}

interface RawOrderDto {
  id: string;
  order_no?: string;
  orderNo?: string;
  site_id?: string;
  siteId?: string;
  site_name?: string;
  siteName?: string;
  cell_id?: string;
  cellId?: string;
  cell_no?: string;
  cellNo?: string;
  size?: string;
  status?: string;
  duration_minutes?: number;
  durationMinutes?: number;
  fee_snapshot?: RawFeeSnapshotDto;
  feeSnapshot?: RawFeeSnapshotDto;
  overdue_fee_fen?: string | number;
  overdueFeeFen?: string | number;
  created_at?: string;
  createdAt?: string;
  paid_at?: string;
  paidAt?: string;
  deposited_at?: string;
  depositedAt?: string;
  expires_at?: string;
  expiresAt?: string;
  completed_at?: string;
  completedAt?: string;
}

function normalizeFeeSnapshot(raw: RawFeeSnapshotDto | undefined): FeeSnapshotDto {
  return {
    rent_fee_fen: toFen(raw?.rent_fee_fen ?? raw?.rentFeeFen),
    deposit_fen: toFen(raw?.deposit_fen ?? raw?.depositFen),
    discount_fen: toFen(raw?.discount_fen ?? raw?.discountFen),
    total_fen: toFen(raw?.total_fen ?? raw?.totalFen)
  };
}

function normalizeOrder(raw: RawOrderDto): OrderDto {
  return {
    id: raw.id,
    order_no: raw.order_no ?? raw.orderNo ?? "",
    site_id: raw.site_id ?? raw.siteId ?? "",
    site_name: raw.site_name ?? raw.siteName ?? "",
    cell_id: raw.cell_id ?? raw.cellId ?? "",
    cell_no: raw.cell_no ?? raw.cellNo ?? "",
    size: raw.size ?? "CELL_SIZE_UNSPECIFIED",
    status: (raw.status ?? "ORDER_STATUS_UNSPECIFIED") as OrderStatus,
    duration_minutes: raw.duration_minutes ?? raw.durationMinutes ?? 0,
    fee_snapshot: normalizeFeeSnapshot(raw.fee_snapshot ?? raw.feeSnapshot),
    overdue_fee_fen: toFen(raw.overdue_fee_fen ?? raw.overdueFeeFen),
    created_at: raw.created_at ?? raw.createdAt ?? "",
    paid_at: raw.paid_at ?? raw.paidAt ?? "",
    deposited_at: raw.deposited_at ?? raw.depositedAt ?? "",
    expires_at: raw.expires_at ?? raw.expiresAt ?? "",
    completed_at: raw.completed_at ?? raw.completedAt ?? ""
  };
}

interface OrderReply {
  order?: RawOrderDto;
}

export function createOrderClient(deps: { request: RequestFn }) {
  const postOrderAction = async (path: string): Promise<OrderDto> => {
    const result = await deps.request<OrderReply>({
      path,
      method: "POST",
      data: {}
    });
    if (!result.order) throw new Error("missing order in reply");
    return normalizeOrder(result.order);
  };

  return {
    createOrder: async (input: {
      siteId: string;
      size: string;
      durationMinutes: number;
    }): Promise<OrderDto> => {
      const result = await deps.request<OrderReply>({
        path: "/v1/orders",
        method: "POST",
        data: {
          site_id: input.siteId,
          size: input.size,
          duration_minutes: input.durationMinutes
        }
      });
      if (!result.order) throw new Error("missing order in reply");
      return normalizeOrder(result.order);
    },
    getOrder: async (orderId: string): Promise<OrderDto> => {
      const result = await deps.request<OrderReply>({
        path: `/v1/orders/${encodeURIComponent(orderId)}`
      });
      if (!result.order) throw new Error("missing order in reply");
      return normalizeOrder(result.order);
    },
    listOrders: async (): Promise<{ orders: OrderDto[] }> => {
      const result = await deps.request<{ orders?: RawOrderDto[] }>({
        path: "/v1/orders"
      });
      return { orders: (result.orders ?? []).map(normalizeOrder) };
    },
    cancelOrder: (orderId: string) =>
      postOrderAction(`/v1/orders/${encodeURIComponent(orderId)}/cancellations`),
    openDepositDoor: (orderId: string) =>
      postOrderAction(`/v1/orders/${encodeURIComponent(orderId)}/deposit-door-requests`),
    openPickupDoor: (orderId: string) =>
      postOrderAction(`/v1/orders/${encodeURIComponent(orderId)}/pickup-door-requests`),
    /** Development simulator only; the backend rejects it outside APP_ENV=development. */
    confirmDevelopmentPayment: (orderId: string) =>
      postOrderAction(
        `/v1/development/orders/${encodeURIComponent(orderId)}/payment-confirmations`
      ),
    /** Development simulator only; the backend rejects it outside APP_ENV=development. */
    settleOverdue: (orderId: string) =>
      postOrderAction(
        `/v1/development/orders/${encodeURIComponent(orderId)}/overdue-payments`
      )
  };
}

export type OrderClient = ReturnType<typeof createOrderClient>;
