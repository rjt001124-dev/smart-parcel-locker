import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FeeSummary } from "./fee-summary";

describe("FeeSummary", () => {
  it("renders rent, deposit, discount and total in yuan from integer fen", () => {
    render(
      <FeeSummary
        rentFeeFen={300}
        depositFen={0}
        discountFen={0}
        totalFen={300}
      />
    );

    expect(screen.getByText("寄存费")).toBeInTheDocument();
    // rent and total are both ¥3.00 in this snapshot
    expect(screen.getAllByText("¥3.00")).toHaveLength(2);
    expect(screen.getByText("押金")).toBeInTheDocument();
    expect(screen.getByText("合计")).toBeInTheDocument();
    expect(screen.getByText("¥0.00")).toBeInTheDocument();
  });

  it("renders overdue fee row only when overdueFeeFen > 0", () => {
    const { rerender } = render(
      <FeeSummary rentFeeFen={500} depositFen={0} discountFen={0} totalFen={500} />
    );
    expect(screen.queryByText("逾期费")).not.toBeInTheDocument();

    rerender(
      <FeeSummary
        rentFeeFen={500}
        depositFen={0}
        discountFen={0}
        totalFen={1300}
        overdueFeeFen={800}
      />
    );
    expect(screen.getByText("逾期费")).toBeInTheDocument();
    expect(screen.getByText("¥8.00")).toBeInTheDocument();
    expect(screen.getByText("¥13.00")).toBeInTheDocument();
  });

  it("renders discount as negative amount when present", () => {
    render(
      <FeeSummary
        rentFeeFen={1000}
        depositFen={0}
        discountFen={200}
        totalFen={800}
      />
    );
    expect(screen.getByText("优惠")).toBeInTheDocument();
    expect(screen.getByText("-¥2.00")).toBeInTheDocument();
  });
});
