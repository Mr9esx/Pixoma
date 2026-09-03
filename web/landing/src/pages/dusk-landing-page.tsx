import HeroSection from "@/components/hero-section-2-hero-section";
import FeaturesThree from "@/components/features-3";
import FeaturesFour from "@/components/features-4";
import Content from "@/components/content-2";
import Stats from "@/components/stats-2";
import FAQs from "@/components/faqs-1";
import InstallSection from "@/components/install-section";
import CallToAction from "@/components/call-to-action-2";
import Footer from "@/components/footer-2";

export default function DuskLandingPage() {
  return (
    <>
      <HeroSection />
      <Content />
      <FeaturesThree />
      <FeaturesFour />
      {/* <Stats /> */}
      <InstallSection />
      <FAQs />
      <CallToAction />
      <Footer />
    </>
  );
}
