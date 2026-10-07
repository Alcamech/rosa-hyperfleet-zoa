package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/accessclient"
	"github.com/spf13/cobra"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/output"
)

// ssmDeploymentsPath is the SSM Parameter Store path that contains
// deployment pointers (deployment_name → Access Function URL + invoker role).
// Written by RC Terraform into the Central Account, read directly by the CLI
// from the SRE's active credentials (Central Account).
const ssmDeploymentsPath = "/zoa/deployments"

// ssmDeployment represents a single deployment entry in SSM.
type ssmDeployment struct {
	DeploymentName string `json:"deployment_name"`
	AccessURL      string `json:"access_url"`
	InvokerRoleARN string `json:"invoker_role_arn"`
	Region         string `json:"region"`
	AccountID      string `json:"account_id"`
}

func newDeploymentsCommand(opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployments",
		Short: "List available ZOA deployments",
		Long: `List all ZOA deployments by reading SSM Parameter Store in the Central account.

No Access Lambda or invoker role — direct SSM only. Does not support
--access-url or --invoker-role-arn.

By default SSM is read in us-east-1; if that fails, the CLI retries eu-west-1.
Pass --deployments-region to use one region only (no automatic failover).

Use DEPLOYMENT names from this table with zoa targets and zoa session.
The default table includes Access URL and invoker role; -o wide adds account ID.`,
		Example: `  zoa deployments

  zoa deployments -o wide
  zoa deployments -o json

  zoa deployments --deployments-region eu-west-1`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if accessclient.DirectAccessOverrideActive(opts.AccessFunctionURL, opts.AccessInvokerRoleARN) {
				return fmt.Errorf("zoa deployments does not support --access-url or --invoker-role-arn (deployment SSM only); use those flags with zoa targets or zoa session")
			}
			if err := accessclient.ValidateDirectAccessPair(opts.AccessFunctionURL, opts.AccessInvokerRoleARN); err != nil {
				return err
			}

			deployments, err := listDeploymentsFromSSM(cmd.Context(), opts.DeploymentsRegion)
			if err != nil {
				return fmt.Errorf("listing deployments from SSM: %w", err)
			}

			if opts.OutputFormat == output.FormatJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]interface{}{
					"items": deployments,
					"count": len(deployments),
				})
			}

			if len(deployments) == 0 {
				fmt.Println("No deployments found in SSM")
				fmt.Println()
				fmt.Println("Hint: ensure AWS credentials can read deployment discovery in this account")
				fmt.Printf("      (SSM path: %s)\n", ssmDeploymentsPath)
				return nil
			}

			return printDeploymentsTable(os.Stdout, opts, deployments)
		},
	}

	return cmd
}

func printDeploymentsTable(w io.Writer, opts *GlobalOptions, deployments []ssmDeployment) error {
	wide := opts.OutputFormat == output.FormatWide
	tw := output.NewTable(w)
	if wide {
		fmt.Fprintln(tw, "DEPLOYMENT\tREGION\tACCESS URL\tINVOKER ROLE\tACCOUNT ID")
	} else {
		fmt.Fprintln(tw, "DEPLOYMENT\tREGION\tACCESS URL\tINVOKER ROLE")
	}
	for _, d := range deployments {
		if wide {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				d.DeploymentName,
				output.Dash(d.Region),
				output.Dash(d.AccessURL),
				output.Dash(d.InvokerRoleARN),
				output.Dash(d.AccountID))
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			d.DeploymentName,
			output.Dash(d.Region),
			output.Dash(d.AccessURL),
			output.Dash(d.InvokerRoleARN))
	}
	return tw.Flush()
}

// listDeploymentsFromSSM reads /zoa/deployments from SSM Parameter Store
// using the SRE's active AWS credentials (direct read, no Lambda involved).
func listDeploymentsFromSSM(ctx context.Context, deploymentsRegion string) ([]ssmDeployment, error) {
	params, err := accessclient.ListDeploymentParameters(ctx, deploymentsRegion)
	if err != nil {
		return nil, err
	}
	return parseSSMDeploymentParameters(params)
}

// parseSSMDeploymentParameters decodes deployment JSON blobs from SSM parameters.
func parseSSMDeploymentParameters(params []ssmtypes.Parameter) ([]ssmDeployment, error) {
	var deployments []ssmDeployment
	for _, param := range params {
		var d ssmDeployment
		if err := json.Unmarshal([]byte(aws.ToString(param.Value)), &d); err != nil {
			return nil, fmt.Errorf("parsing SSM parameter %s: %w", aws.ToString(param.Name), err)
		}
		deployments = append(deployments, d)
	}
	return deployments, nil
}
