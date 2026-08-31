import { useTranslation } from "react-i18next";
import { SectionShell } from "@/components/SectionShell";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { faqIds } from "@/constants/content";

export function FaqSection() {
  const { t } = useTranslation();

  return (
    <SectionShell id="faq" title={t("faq.title")} className="bg-muted/40">
      <Accordion
        type="single"
        collapsible
        defaultValue="install"
        className="mx-auto flex max-w-3xl flex-col gap-4"
      >
        {faqIds.map((id) => (
          <AccordionItem key={id} value={id} className="faq-card px-5">
            <AccordionTrigger className="min-h-14 py-4 text-base hover:no-underline">
              {t(`faq.${id}.question`)}
            </AccordionTrigger>
            <AccordionContent className="pb-5 leading-7 text-muted-foreground">
              {t(`faq.${id}.answer`)}
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>
    </SectionShell>
  );
}
