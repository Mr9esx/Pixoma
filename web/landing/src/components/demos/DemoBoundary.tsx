import { Component, type ReactNode } from "react";

type DemoBoundaryProps = {
  children: ReactNode;
  fallback: ReactNode;
};

type DemoBoundaryState = {
  failed: boolean;
};

export class DemoBoundary extends Component<
  DemoBoundaryProps,
  DemoBoundaryState
> {
  state: DemoBoundaryState = { failed: false };

  static getDerivedStateFromError(): DemoBoundaryState {
    return { failed: true };
  }

  override render() {
    return this.state.failed ? this.props.fallback : this.props.children;
  }
}
