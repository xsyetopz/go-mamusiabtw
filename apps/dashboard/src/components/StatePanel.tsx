import { Alert, Card, Code, Group, Stack, Text, Title } from "@mantine/core";
import type { ReactNode } from "react";

type Props = {
	title: string;
	status: ReactNode;
	children: ReactNode;
};

export function StatePanel({ title, status, children }: Props) {
	return (
		<Card className="panel-card" withBorder={true}>
			<Stack gap="sm">
				<Group justify="space-between">
					<Title order={3}>{title}</Title>
					{status}
				</Group>
				{children}
			</Stack>
		</Card>
	);
}

type MessageProps = {
	title: string;
	children: ReactNode;
};

export function SetupMessage({ title, children }: MessageProps) {
	return (
		<Alert color="gray" radius="md" variant="light" title={title}>
			{children}
		</Alert>
	);
}

type PathProps = {
	label: string;
	value: string;
};

export function CodeLine({ label, value }: PathProps) {
	return (
		<Text size="sm">
			{label}: <Code>{value}</Code>
		</Text>
	);
}
