import { useRouter } from "vue-router";

export function useRichText() {
  const router = useRouter();

  const formatRichText = (text: string): string => {
    if (!text) return "";
    
    // Replace hashtags with clickable blue spans
    let formatted = text.replace(
      /#(\w+)/g, 
      "<span class=\"rich-text-hashtag\" data-hashtag=\"$1\">#$1</span>"
    );
    
    // Replace mentions with clickable blue spans
    formatted = formatted.replace(
      /@(\w+)/g, 
      "<span class=\"rich-text-mention\" data-username=\"$1\">@$1</span>"
    );
    
    return formatted;
  };

  const handleRichTextClick = (event: MouseEvent) => {
    const target = event.target as HTMLElement;
    
    // Handle hashtag click
    if (target.classList.contains("rich-text-hashtag")) {
      const hashtag = target.getAttribute("data-hashtag");
      if (hashtag) {
        router.push(`/explore/tags/${hashtag}`);
      }
    }
    
    // Handle mention click
    if (target.classList.contains("rich-text-mention")) {
      const username = target.getAttribute("data-username");
      if (username) {
        router.push(`/profile/${username}`);
      }
    }
  };

  return {
    formatRichText,
    handleRichTextClick
  };
}
