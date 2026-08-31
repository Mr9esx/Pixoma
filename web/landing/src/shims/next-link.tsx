import type { AnchorHTMLAttributes } from "react";

type NextLinkProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  href?: string;
};

export default function NextLink({ href, children, ...props }: NextLinkProps) {
  return (
    <a href={href} {...props}>
      {children}
    </a>
  );
}
