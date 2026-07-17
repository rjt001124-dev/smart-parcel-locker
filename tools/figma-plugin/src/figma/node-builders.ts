export interface AutoLayoutSpec {
  direction: "HORIZONTAL" | "VERTICAL";
  gap: number;
  padding: number;
  width?: number;
  height?: number;
}

export interface TextSpec {
  text: string;
  fontFamily: string;
  fontStyle: string;
  fontSize: number;
  lineHeight: number;
  colorToken: string;
}

export interface ShapeSpec {
  fillToken: string;
  strokeToken?: string;
  radiusToken: string;
}

export interface ComponentFamilySpec {
  key: string;
  name: string;
  variants: readonly Record<string, string>[];
  layout: AutoLayoutSpec;
}

export interface ScreenSpec {
  key: string;
  name: string;
  width: number;
  height: number;
  layout: AutoLayoutSpec;
  sections: readonly string[];
}
