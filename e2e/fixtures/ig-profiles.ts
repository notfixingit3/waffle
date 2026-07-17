/**
 * Instagram in-app browser profiles for Playwright projects.
 *
 * UA strings sourced from live IG app captures (see plan
 * .omo/plans/ig-browser-claim-check.md todo 3). UA alone at a desktop
 * viewport would miss the mobile-first layout under test
 * (`grid-cols-5 sm:grid-cols-10` breakpoint), so each UA is paired with
 * the device metrics of the phone it was captured on.
 */

export interface IgDeviceMetrics {
  readonly viewport: { readonly width: number; readonly height: number };
  readonly deviceScaleFactor: number;
  readonly isMobile: boolean;
  readonly hasTouch: boolean;
}

export interface IgProfile {
  readonly userAgent: string;
  readonly deviceMetrics: IgDeviceMetrics;
}

export const igIosProfile: IgProfile = {
  userAgent:
    'Mozilla/5.0 (iPhone; CPU iPhone OS 26_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/23F77 Instagram 428.2.0.37.66 (iPhone16,2; iOS 26_5; en_US; en; scale=3.00; 1290x2796; IABMV/1; 961927775) NW/3 Safari/604.1',
  deviceMetrics: {
    viewport: { width: 393, height: 852 },
    deviceScaleFactor: 3,
    isMobile: true,
    hasTouch: true,
  },
};

export const igAndroidProfile: IgProfile = {
  userAgent:
    'Mozilla/5.0 (Linux; Android 16; SM-S911B Build/BP2A.250605.031.A3; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/147.0.7727.55 Mobile Safari/537.36 Instagram 425.0.0.47.61 Android (36/16; 450dpi; 1080x2340; samsung; SM-S911B; dm1q; qcom; de_DE; 938256486; IABMV/1)',
  deviceMetrics: {
    viewport: { width: 360, height: 780 },
    deviceScaleFactor: 3,
    isMobile: true,
    hasTouch: true,
  },
};
