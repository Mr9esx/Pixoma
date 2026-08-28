import { useEffect } from "react";
import { Navigate, useParams } from "react-router-dom";
import i18n from "@/i18n";
import { isLangSegment, langSegmentToLocale } from "@/router/segments";
import { CTASection } from "@/components/sections/CTASection";
import { FeaturesSection } from "@/components/sections/FeaturesSection";
import { HeroSection } from "@/components/sections/HeroSection";
import { UseCasesSection } from "@/components/sections/UseCasesSection";
import { Header } from "./Header";
import { Footer } from "./Footer";

export function LangLayout() {
  const { lang } = useParams();
  const locale = langSegmentToLocale[lang ?? "cn"] ?? "zh-CN";
  const valid = isLangSegment(lang);

  useEffect(() => {
    if (!valid) return;
    void i18n.changeLanguage(locale);
    document.documentElement.lang = locale;
  }, [valid, locale]);

  if (!valid) {
    return <Navigate to="/cn" replace />;
  }

  return (
    <div className="flex min-h-svh flex-col bg-background text-foreground">
      <Header />
      <main className="flex-1">
        <HeroSection />
        <FeaturesSection />
        <UseCasesSection />
        <CTASection />
      </main>
      <Footer />
    </div>
  );
}
