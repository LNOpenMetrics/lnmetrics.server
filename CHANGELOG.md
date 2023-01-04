# v0.0.1-rc7

## Fixed
- bugs when we will go cu calculate the metrics ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/86c470c289f7a112553992d1ca6c7ac228a59c2a)). @vincenzopalazzo 14-12-2022
- check only for bitcoin network ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/189f00a093ac51aedb9d56b955363b9419999bff)). @vincenzopalazzo 11-12-2022
- command line parser bug ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/b6361c225483bb22de365f8a68012657f94ea3cd)). @vincenzopalazzo 11-12-2022

## Added
- implement a recurrent clean up for the db ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/cc320d10e1b1f51e7799fd785ff6ef27a6b52878)). @vincenzopalazzo 14-12-2022
- add cli args parser ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/e98661fb291c790aa4fbc05f98b432c0e34b938b)). @vincenzopalazzo 11-12-2022
- complete migration to the multiple db ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/4996f7d6b38ddc1c13199b2e83abc5d2f9ec4b00)). @vincenzopalazzo 11-12-2022
- support different database for different network ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/fd6df5bd62ba1f9839058dc07a95416369317f2e)). @vincenzopalazzo 10-12-2022


# v0.0.1-rc6

## Fixed
- add check if the timestamp is invalid set the actual timestamp o… ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/ad3c2199d7ab1c3e858bdd806a1adc1dafc459ef)). @vincenzopalazzo 30-08-2022
- error handling inside the lnmetrics.utils ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/3c52ab6d3758ee7df2d339faa3b2040788faa04d)). @vincenzopalazzo 30-08-2022
- hash the message to verify ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/97e8fdd877bfe287efaacbd2db705e244a5e4a2c)). @vincenzopalazzo 29-08-2022

## Added
- remove the rest backend is the url is not specified in favor of… ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/f9caa204e63c3e78aae3ce6b1bf96d3128bd3695)). @vincenzopalazzo 29-08-2022


# v0.0.1-rc5

## Fixed
- calculate the channel metrics based on the server timestamp ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/e43f86f289cc3fb01dac24e1d01812d2f71b2ba5)). @vincenzopalazzo 29-08-2022


# v0.0.1-rc4

## Fixed
- clean up async code for better time ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/9fadf94167662bf074d0f78ae418bfc5b7c44150)). @vincenzopalazzo 28-08-2022
- typo in the paginator pattern hash_next_page instead of has_next ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/99fc9927d3dd191b784af5516a365431b1e2a98c)). @vincenzopalazzo 28-08-2022

## Added
- upgrade dependeces ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/dd2e0f43b30c5cdb6cefa90df2c7fee8cb30e815)). @vincenzopalazzo 28-08-2022


# v0.0.1-rc3

## Fixed
- move the json analysis sync to see if some bugs are gone ([commit](https://github.com/LNOpenMetrics/lnmetrics.server/commit/e6df88b2a7c548b3e942123131e4a410b5de9c9b)). @vincenzopalazzo 19-08-2022
