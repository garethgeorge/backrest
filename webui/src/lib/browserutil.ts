export const isMobile = () => {
  // check if window is narrow 
  if (window.innerWidth <= 1024) {
    return true;
  }
  // check if user agent is mobile
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
}