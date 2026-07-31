import { Text, View } from "@tarojs/components";
import { toOrderStatusView, type OrderStatus } from "@spl/domain-ui/order";

import "./status-timeline.scss";

export interface StatusTimelineProps {
  status: OrderStatus;
}

const MAIN_FLOW: OrderStatus[] = [
  "ORDER_STATUS_PENDING_PAYMENT",
  "ORDER_STATUS_AWAITING_DEPOSIT",
  "ORDER_STATUS_IN_STORAGE",
  "ORDER_STATUS_AWAITING_PICKUP",
  "ORDER_STATUS_COMPLETED"
];

/** PAID is transient and displayed at the same step as AWAITING_DEPOSIT. */
const STATUS_STEP_INDEX: Partial<Record<OrderStatus, number>> = {
  ORDER_STATUS_PENDING_PAYMENT: 0,
  ORDER_STATUS_PAID: 1,
  ORDER_STATUS_AWAITING_DEPOSIT: 1,
  ORDER_STATUS_IN_STORAGE: 2,
  ORDER_STATUS_AWAITING_PICKUP: 3,
  ORDER_STATUS_COMPLETED: 4
};

type StepState = "done" | "current" | "upcoming";

function stepClass(state: StepState): string {
  return `status-timeline__label status-timeline__label--${state}`;
}

/**
 * Visualizes the server-driven order lifecycle. Overdue orders show an extra
 * branch step; cancelled orders collapse to a single terminal step.
 */
export function StatusTimeline({ status }: StatusTimelineProps) {
  if (status === "ORDER_STATUS_CANCELLED") {
    return (
      <View className="status-timeline">
        <View className="status-timeline__step">
          <Text className={stepClass("current")}>已取消</Text>
        </View>
      </View>
    );
  }

  const isOverdue = status === "ORDER_STATUS_OVERDUE";
  // Overdue happens after IN_STORAGE; steps up to IN_STORAGE are done.
  const currentIndex = isOverdue ? 2 : STATUS_STEP_INDEX[status] ?? 0;

  return (
    <View className="status-timeline">
      {MAIN_FLOW.map((step, index) => {
        const view = toOrderStatusView(step);
        let state: StepState;
        if (isOverdue) {
          state = index <= currentIndex ? "done" : "upcoming";
        } else if (index < currentIndex) {
          state = "done";
        } else if (index === currentIndex) {
          state = "current";
        } else {
          state = "upcoming";
        }
        return (
          <View className="status-timeline__step" key={step}>
            <Text className={stepClass(state)}>{view.label}</Text>
            {isOverdue && index === currentIndex ? (
              <Text className={stepClass("current")}>已逾期</Text>
            ) : null}
          </View>
        );
      })}
    </View>
  );
}
