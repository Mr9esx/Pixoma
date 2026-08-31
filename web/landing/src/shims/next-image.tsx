import type { ComponentProps } from "react";

type NextImageProps = ComponentProps<"img">;

export default function NextImage(props: NextImageProps) {
  return <img {...props} />;
}
