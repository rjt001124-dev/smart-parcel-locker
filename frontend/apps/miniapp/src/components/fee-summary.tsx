import { Text, View } from "@tarojs/components";
import { formatFen } from "@spl/domain-ui/money";

import "./fee-summary.scss";

export interface FeeSummaryProps {
  rentFeeFen: number;
  depositFen: number;
  discountFen: number;
  totalFen: number;
  overdueFeeFen?: number;
}

interface RowProps {
  label: string;
  value: string;
  emphasized?: boolean;
}

function Row({ label, value, emphasized = false }: RowProps) {
  return (
    <View className={`fee-summary__row${emphasized ? " fee-summary__row--total" : ""}`}>
      <Text className="fee-summary__label">{label}</Text>
      <Text className="fee-summary__value">{value}</Text>
    </View>
  );
}

/**
 * Renders an order fee snapshot. All amounts are integer fen provided by the
 * server; the component never computes totals itself.
 */
export function FeeSummary({
  rentFeeFen,
  depositFen,
  discountFen,
  totalFen,
  overdueFeeFen = 0
}: FeeSummaryProps) {
  return (
    <View className="fee-summary">
      <Row label="寄存费" value={formatFen(rentFeeFen)} />
      <Row label="押金" value={formatFen(depositFen)} />
      {discountFen > 0 ? (
        <Row label="优惠" value={`-${formatFen(discountFen)}`} />
      ) : null}
      {overdueFeeFen > 0 ? (
        <Row label="逾期费" value={formatFen(overdueFeeFen)} />
      ) : null}
      <Row label="合计" value={formatFen(totalFen)} emphasized />
    </View>
  );
}
