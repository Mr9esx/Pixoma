export const navigationItems = [
  { id: "features", labelKey: "nav.features", href: "#features" },
  { id: "devices", labelKey: "nav.devices", href: "#devices" },
  { id: "audience", labelKey: "nav.audience", href: "#audience" },
  { id: "faq", labelKey: "nav.faq", href: "#faq" },
  { id: "download", labelKey: "nav.download", href: "#download" },
] as const;

export type NavigationItem = (typeof navigationItems)[number];
