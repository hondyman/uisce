// Auto-generated ambient module declarations to silence third-party type missing errors
// This is a temporary pragmatic fix; proper types should be added for production.

declare module '*';

// Generic Radix UI primitives
declare module '@radix-ui/*' { const anyExport: any; export = anyExport; }
declare module '@radix-ui/react-*' { const anyExport: any; export = anyExport; }

declare module 'embla-carousel-react' { export type UseEmblaCarouselType = any; const anyExport: any; export default anyExport; }
declare module 'react-day-picker' { export const DayPicker: any; export default any; }
declare module 'cmdk' { export const Command: any; export default Command; }
declare module 'vaul' { export const Drawer: any; export default Drawer; }
declare module 'sonner' { export const Toaster: any; export const toast: any; export default Toaster; }
declare module 'input-otp' { export const OTPInput: any; export const OTPInputContext: any; export default OTPInput; }
declare module 'react-resizable-panels' { const anyExport: any; export = anyExport; }
declare module '@tabler/icons-react' {
	import * as React from 'react';

	// More precise icon component type used by the frontend.
	export type IconComponent = React.FC<React.SVGProps<SVGSVGElement> & {
		size?: number | string;
		className?: string;
	}>;

	export const IconX: IconComponent;
	export const IconBrain: IconComponent;
	export const IconDeviceFloppy: IconComponent;
		export const IconCube: IconComponent;
		export const IconLock: IconComponent;
		export const IconEye: IconComponent;
	export const IconPlayerPlay: IconComponent;
	export const IconPlugConnected: IconComponent;
	export const IconBook: IconComponent;
	export const IconSelect: IconComponent;
	export const IconChartBar: IconComponent;
	export const IconStack3: IconComponent;
	export const IconCheck: IconComponent;
	export const IconAlertTriangle: IconComponent;
	export const IconChevronDown: IconComponent;
	export const IconChevronRight: IconComponent;
	export const IconEdit: IconComponent;
	export const IconSearch: IconComponent;
	export const IconTextSize: IconComponent;
	export const IconNumbers: IconComponent;
	export const IconCalendar: IconComponent;
	export const IconToggleLeft: IconComponent;
	export const IconHash: IconComponent;
	export const IconFileText: IconComponent;
	export const IconPlus: IconComponent;
	export const IconSettings: IconComponent;
			export const IconUser: IconComponent;
			export const IconLogout: IconComponent;
	export const IconCode: IconComponent;
	export const IconDatabase: IconComponent;
	export const IconDownload: IconComponent;
	export const IconCopy: IconComponent;
	export const IconTrash: IconComponent;
	export const IconFilter: IconComponent;
	export const IconRulerMeasure: IconComponent;

	const _default: { [key: string]: IconComponent };
	export default _default;
}

declare module 'react-dnd' { export const DndProvider: any; export const useDrag: any; export const useDrop: any; export default any; }
declare module 'react-dnd-html5-backend' { const HTML5Backend: any; export default HTML5Backend; export { HTML5Backend }; }

declare module '@radix-ui/react-dialog' {
	export const Root: any;
	export const Trigger: any;
	export const Portal: any;
	export const Overlay: any;
	export const Content: any;
	export const Title: any;
	export const Description: any;
	export const Close: any;
	export const RootProps: any;
	export const Dialog: any;
	export default Dialog;
}

// Fallback for any other missing module
declare module '*';
