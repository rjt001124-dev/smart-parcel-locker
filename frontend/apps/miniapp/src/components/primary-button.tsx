import { Button } from "@tarojs/components";
import "./primary-button.scss";

export interface PrimaryButtonProps {
  label: string;
  loading?: boolean;
  onClick: () => void;
}

export function PrimaryButton({ label, loading = false, onClick }: PrimaryButtonProps) {
  return (
    <Button
      className={`primary-button${loading ? " primary-button--disabled" : ""}`}
      disabled={loading}
      onClick={onClick}
    >
      {loading ? "处理中…" : label}
    </Button>
  );
}
